package heaphelper

import (
	"context"
	"errors"
	"os"
	"sync"
	"time"

	"github.com/WangWilly/xSync/pkgs/clients/twitterclient"
	"github.com/WangWilly/xSync/pkgs/downloading/dtos/smartpathdto"
	"github.com/WangWilly/xSync/pkgs/model"
	"github.com/WangWilly/xSync/pkgs/repos/listrepo"
	"github.com/WangWilly/xSync/pkgs/repos/userrepo"
	"github.com/WangWilly/xSync/pkgs/tasks"
	"github.com/WangWilly/xSync/pkgs/utils"
	"github.com/jmoiron/sqlx"

	log "github.com/sirupsen/logrus"
)

type helper struct {
	uidToUserMap         map[uint64]*twitterclient.User
	users                []UserWithinListEntity
	userSmartPathToDepth map[*smartpathdto.UserSmartPath]int

	userRepo             UserRepo
	listRepo             ListRepo
	twitterClientManager *twitterclient.Manager

	syncedUserSmartPaths *utils.SyncMap[uint64, *smartpathdto.UserSmartPath]
	syncedListToUsersMap *utils.SyncMap[int, *utils.SyncMap[uint64, struct{}]]

	heap *utils.Heap[*smartpathdto.UserSmartPath]

	mtx sync.Mutex
}

func NewHelperFromTasks(
	ctx context.Context,
	client *twitterclient.Client,
	db *sqlx.DB,
	task *tasks.Task,
	rootDir string,
	twitterClientManager *twitterclient.Manager,
) (*helper, error) {
	res := &helper{
		uidToUserMap:         nil,
		users:                nil,
		userSmartPathToDepth: make(map[*smartpathdto.UserSmartPath]int),

		userRepo:             userrepo.New(),
		listRepo:             listrepo.New(),
		twitterClientManager: twitterClientManager,

		syncedUserSmartPaths: utils.NewSyncMap[uint64, *smartpathdto.UserSmartPath](),
		syncedListToUsersMap: utils.NewSyncMap[int, *utils.SyncMap[uint64, struct{}]](),

		heap: nil,
		mtx:  sync.Mutex{},
	}

	usersWithinListEntity, err := res.getUsersWithinListEntity(ctx, client, db, task, rootDir)
	if err != nil || len(usersWithinListEntity) == 0 {
		return nil, errors.New("failed to get users within list entity: " + err.Error())
	}
	res.users = usersWithinListEntity
	res.uidToUserMap = make(map[uint64]*twitterclient.User, len(usersWithinListEntity))
	for _, u := range usersWithinListEntity {
		res.uidToUserMap[u.User.TwitterId] = u.User
	}

	return res, nil
}

/**
func NewHelper(users []UserWithinListEntity, twitterClientManager *twitterclient.Manager) *helper {
	uidToUserMap := make(map[uint64]*twitterclient.User)
	for _, u := range users {
		uidToUserMap[u.User.TwitterId] = u.User
	}

	return &helper{
		uidToUserMap:         uidToUserMap,
		users:                users,
		userSmartPathToDepth: make(map[*smartpathdto.UserSmartPath]int),

		listRepo:             listrepo.New(),
		twitterClientManager: twitterClientManager,

		syncedUserSmartPaths: sync.Map{},
		syncedListUsers:      sync.Map{},

		heap: nil,
		mtx:  sync.Mutex{},
	}
}
*/

////////////////////////////////////////////////////////////////////////////////

func (h *helper) MakeHeap(
	ctx context.Context,
	db *sqlx.DB,
	dir string,
	autoFollow bool,
) error {
	h.mtx.Lock()
	defer h.mtx.Unlock()

	if h.heap != nil {
		return nil
	}

	ctx, cancel := context.WithCancelCause(ctx)
	defer func() {
		utils.PanicHandler(cancel)
	}()

	logger := log.WithField("worker", "updating")
	logger.Infoln("start pre processing users")

	tic := time.Now()
	debugDeepest := 0
	debugMissingTweets := 0

	userSmartPathList := make([]*smartpathdto.UserSmartPath, 0)
	for _, userWithinList := range h.users {
		user := userWithinList.User

		userLogger := logger.WithField("user", user.Title())
		userLogger.Infoln("processing user")

		if isIngoreUser(user) {
			userLogger.Infoln("user is ignored, skipping")
			continue
		}

		userSmartPath := h.getUserSmartPathByTwitterId(user, db, dir)
		if userSmartPath == nil {
			userLogger.Warnln("failed to get user smart path, skipping user")
			continue
		}
		userSmartPathList = append(userSmartPathList, userSmartPath)

		// 计算深度
		if user.MediaCount != 0 && user.IsVisiable() {
			debugMissingTweets += max(0, user.MediaCount-int(userSmartPath.Record.MediaCount.Int32))
			h.userSmartPathToDepth[userSmartPath] = calcUserDepth(
				int(userSmartPath.Record.MediaCount.Int32),
				user.MediaCount,
			)
			debugDeepest = max(debugDeepest, h.userSmartPathToDepth[userSmartPath])
		}

		h.doFollow(ctx, user, autoFollow)

		// 即便同步一个用户时也同步了所有指向此用户的链接，
		// 但此用户仍可能会是一个新的 “列表-用户”，所以判断此用户链接是否同步过，
		// 如果否，那么创建一个属于此列表的用户链接
		ListId := userWithinList.MaybeListId
		if ListId == nil {
			userLogger.Infoln("list entity ID is nil, skipping user")
			continue
		}

		userTwitterIdSet, _ := h.syncedListToUsersMap.LoadOrStore(
			*ListId,
			utils.NewSyncMap[uint64, struct{}](),
		)
		_, loaded := userTwitterIdSet.LoadOrStore(user.TwitterId, struct{}{})
		if loaded {
			userLogger.Infoln("user already exists in list, skipping")
			continue
		}

		// 为当前列表的新用户创建符号链接
		userLogger.Infoln("creating link for user in list")
		userLink := &model.UserLink{
			Name:                 userSmartPath.Name(),
			ListEntityIdBelongTo: int32(*ListId),
			UserTwitterId:        user.TwitterId,
		}
		linkpath, err := userLink.Path(db)
		if err != nil {
			userLogger.Warnln("failed to create link for user:", err)
			continue
		}
		storageFolderForUser, _ := userSmartPath.Path()
		if err := os.Symlink(storageFolderForUser, linkpath); err == nil || os.IsExist(err) {
			err := h.userRepo.CreateLink(db, userLink)
			if err != nil {
				userLogger.Warnln("failed to create link in database:", err)
			}
		}
	}

	lessFunc := func(lhs, rhs *smartpathdto.UserSmartPath) bool {
		luser, ruser := h.uidToUserMap[lhs.TwitterId()], h.uidToUserMap[rhs.TwitterId()]
		lOnlyMater := luser.IsProtected && luser.Followstate == twitterclient.FS_FOLLOWING
		rOnlyMaster := ruser.IsProtected && ruser.Followstate == twitterclient.FS_FOLLOWING

		if lOnlyMater == rOnlyMaster {
			return h.userSmartPathToDepth[lhs] > h.userSmartPathToDepth[rhs]
		}
		return lOnlyMater // 优先让 master 获取只有他能看到的
	}
	userUserSmartPathHeap := utils.NewByHeapify(userSmartPathList, lessFunc)
	if userUserSmartPathHeap.Empty() {
		logger.Infoln("no user to process")
		return errors.New("no user to process")
	}

	logger.Debugln("preprocessing finish, elapsed:", time.Since(tic))
	logger.Debugln("real members:", userUserSmartPathHeap.Size())
	logger.Debugln("missing tweets:", debugMissingTweets)
	logger.Debugln("deepest:", debugDeepest)

	h.heap = userUserSmartPathHeap
	return nil
}

func (h *helper) getUserSmartPathByTwitterId(
	user *twitterclient.User,
	db *sqlx.DB,
	dir string,
) *smartpathdto.UserSmartPath {
	logger := log.
		WithField("caller", "heaphelper.getUserSmartPathByTwitterId").
		WithField("user", user.Title())

	userSmartPath, loaded := h.syncedUserSmartPaths.Load(user.TwitterId)
	if loaded {
		logger.Infoln("user found in syncedUserSmartPaths, using existing smart path")
		return userSmartPath
	}

	var err error
	userSmartPath, err = syncUserToDbAndGetSmartPath(db, user, dir)
	if err != nil {
		logger.Warnln("failed to update user or entity", err)
		return nil
	}
	h.syncedUserSmartPaths.Store(user.TwitterId, userSmartPath)

	// 同步所有现存的指向此用户的符号链接
	linksUserBelongTo, err := h.userRepo.GetLinks(db, user.TwitterId)
	if err != nil {
		logger.Warnln("failed to get links to user:", err)
		return userSmartPath
	}

	inStoragePath, _ := userSmartPath.Path()
	for _, userLink := range linksUserBelongTo {
		logger.
			WithField("userLink", userLink.Name).
			Infoln("updating userLink that belongs to the user")

		if err = updateUserLink(userLink, db, inStoragePath); err != nil {
			logger.Warnln("failed to update link:", err)
		}

		userTwitterIdSet, _ := h.syncedListToUsersMap.LoadOrStore(
			int(userLink.ListEntityIdBelongTo),
			utils.NewSyncMap[uint64, struct{}](),
		)
		userTwitterIdSet.Store(user.TwitterId, struct{}{})
	}

	return userSmartPath
}

func (h *helper) doFollow(
	ctx context.Context,
	user *twitterclient.User,
	autoFollow bool,
) {
	logger := log.
		WithField("caller", "heaphelper.doFollowOrNot").
		WithField("user", user.Title())

	client := h.twitterClientManager.GetMasterClient()

	// 自动关注
	if user.IsProtected && user.Followstate == twitterclient.FS_UNFOLLOW && autoFollow {
		logger.Infoln("user is protected and not followed, trying to follow")

		if err := client.FollowUser(ctx, user.TwitterId); err != nil {
			logger.Warnln("failed to follow user:", err)
			return
		}
		logger.Debugln("follow request has been sent")
	}
}

////////////////////////////////////////////////////////////////////////////////

func (h *helper) GetHeap() *utils.Heap[*smartpathdto.UserSmartPath] {
	h.mtx.Lock()
	defer h.mtx.Unlock()

	if h.heap == nil {
		panic("heap is not initialized, call MakeHeap first")
	}
	return h.heap
}

func (h *helper) GetDepth(userSmartPath *smartpathdto.UserSmartPath) int {
	h.mtx.Lock()
	defer h.mtx.Unlock()

	if depth, ok := h.userSmartPathToDepth[userSmartPath]; ok {
		return depth
	}
	return 0
}

func (h *helper) GetUserByTwitterId(twitterId uint64) *twitterclient.User {
	h.mtx.Lock()
	defer h.mtx.Unlock()

	if user, ok := h.uidToUserMap[twitterId]; ok {
		return user
	}
	return nil
}
