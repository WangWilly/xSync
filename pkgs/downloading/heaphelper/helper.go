package heaphelper

import (
	"context"
	"errors"
	"os"
	"sync"
	"time"

	"github.com/WangWilly/xSync/pkgs/clients/twitterclient"
	"github.com/WangWilly/xSync/pkgs/database"
	"github.com/WangWilly/xSync/pkgs/downloading/dtos/smartpathdto"
	"github.com/WangWilly/xSync/pkgs/utils"
	"github.com/jmoiron/sqlx"

	log "github.com/sirupsen/logrus"
)

type helper struct {
	uidToUserMap         map[uint64]*twitterclient.User
	users                []UserWithinListEntity
	userSmartPathToDepth map[*smartpathdto.UserSmartPath]int

	twitterClientManager *twitterclient.Manager

	syncedUserSmartPaths sync.Map // map[uint64]*smartpathdto.UserSmartPath
	syncedListUsers      sync.Map // map[int]*sync.Map, where int is ListEntityId, and *sync.Map is map[uint64]struct{}

	heap *utils.Heap[*smartpathdto.UserSmartPath]

	mtx sync.Mutex
}

func NewHelper(users []UserWithinListEntity, twitterClientManager *twitterclient.Manager) *helper {
	uidToUserMap := make(map[uint64]*twitterclient.User)
	for _, u := range users {
		uidToUserMap[u.User.TwitterId] = u.User
	}

	return &helper{
		uidToUserMap:         uidToUserMap,
		users:                users,
		userSmartPathToDepth: make(map[*smartpathdto.UserSmartPath]int),

		twitterClientManager: twitterClientManager,

		syncedUserSmartPaths: sync.Map{},
		syncedListUsers:      sync.Map{},

		heap: nil,
		mtx:  sync.Mutex{},
	}
}

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
	logger := log.WithField("worker", "updating")
	logger.Infoln("start pre processing users")

	defer func() {
		utils.PanicHandler(cancel)
	}()

	tic := time.Now()
	client := h.twitterClientManager.GetMasterClient()
	debugDeepest := 0
	debugMissingTweets := 0
	userUserSmartPathRaw := make([]*smartpathdto.UserSmartPath, 0)
	for _, userWithinList := range h.users {
		user := userWithinList.User
		logger.WithField("user", user.Title()).Infoln("processing user")
		if IsIngoreUser(user) {
			continue
		}

		var userSmartPath *smartpathdto.UserSmartPath
		maybeUserSmartPath, loaded := h.syncedUserSmartPaths.Load(user.TwitterId)
		if !loaded {
			logger.WithField("user", user.Title()).Infoln("user not found in syncedUserSmartPaths, syncing...")

			var err error
			userSmartPath, err = SyncUserToDbAndGetSmartPath(db, user, dir)
			if err != nil {
				logger.WithField("user", user.Title()).Warnln("failed to update user or entity", err)
				continue
			}
			h.syncedUserSmartPaths.Store(user.TwitterId, userSmartPath)

			// 同步所有现存的指向此用户的符号链接
			linkds, err := database.GetUserLinks(db, user.TwitterId)
			if err != nil {
				logger.WithField("user", user.Title()).Warnln("failed to get links to user:", err)
			}
			upath, _ := userSmartPath.Path()
			for _, linkd := range linkds {
				logger.WithField("link", linkd.Name).Infoln("updating link for user")

				if err = UpdateUserLink(linkd, db, upath); err != nil {
					logger.WithField("user", user.Title()).Warnln("failed to update link:", err)
				}
				sl, _ := h.syncedListUsers.LoadOrStore(int(linkd.ParentLstEntityId), &sync.Map{})
				syncedList := sl.(*sync.Map)
				syncedList.Store(user.TwitterId, struct{}{})
			}

			// 计算深度
			if user.MediaCount != 0 && user.IsVisiable() {
				debugMissingTweets += max(0, user.MediaCount-int(userSmartPath.Record.MediaCount.Int32))
				h.userSmartPathToDepth[userSmartPath] = CalcUserDepth(int(userSmartPath.Record.MediaCount.Int32), user.MediaCount)
				// userUserSmartPathHeap.Push(userSmartPath)
				userUserSmartPathRaw = append(userUserSmartPathRaw, userSmartPath)
				debugDeepest = max(debugDeepest, h.userSmartPathToDepth[userSmartPath])
			}

			// 自动关注
			if user.IsProtected && user.Followstate == twitterclient.FS_UNFOLLOW && autoFollow {
				logger.WithField("user", user.Title()).Infoln("user is protected and not followed, trying to follow")

				if err := client.FollowUser(ctx, user.TwitterId); err != nil {
					logger.WithField("user", user.Title()).Warnln("failed to follow user:", err)
				} else {
					logger.WithField("user", user.Title()).Debugln("follow request has been sent")
				}
			}
		} else {
			logger.WithField("user", user.Title()).Infoln("user found in syncedUserSmartPaths, using existing smart path")
			userSmartPath = maybeUserSmartPath.(*smartpathdto.UserSmartPath)
		}

		// 即便同步一个用户时也同步了所有指向此用户的链接，
		// 但此用户仍可能会是一个新的 “列表-用户”，所以判断此用户链接是否同步过，
		// 如果否，那么创建一个属于此列表的用户链接
		logger.WithField("user", user.Title()).Infoln("checking if user is in list")
		leid := userWithinList.Leid
		if leid == nil {
			logger.WithField("user", user.Title()).Infoln("list entity ID is nil, skipping user")
			continue
		}
		sl, _ := h.syncedListUsers.LoadOrStore(*leid, &sync.Map{})
		syncedList := sl.(*sync.Map)
		_, loaded = syncedList.LoadOrStore(user.TwitterId, struct{}{})
		if loaded {
			logger.WithField("user", user.Title()).Infoln("user already exists in list, skipping")
			continue
		}

		// 为当前列表的新用户创建符号链接
		logger.WithField("user", user.Title()).Infoln("creating link for user in list")
		upath, _ := userSmartPath.Path()
		var linkname = userSmartPath.Name()

		curlink := &database.UserLink{}
		curlink.Name = linkname
		curlink.ParentLstEntityId = int32(*leid)
		curlink.Uid = user.TwitterId

		linkpath, err := curlink.Path(db)
		if err == nil {
			if err = os.Symlink(upath, linkpath); err == nil || os.IsExist(err) {
				err = database.CreateUserLink(db, curlink)
			}
		}
		if err != nil {
			logger.WithField("user", user.Title()).Warnln("failed to create link for user:", err)
		}
		logger.WithField("user", user.Title()).Infoln("link created successfully")
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
	userUserSmartPathHeap := utils.NewByHeapify(userUserSmartPathRaw, lessFunc)
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
