package downloading

import (
	"context"
	"runtime"

	"github.com/WangWilly/xSync/pkgs/downloading/dtos/dldto"
	"github.com/WangWilly/xSync/pkgs/downloading/heaphelper"
	"github.com/WangWilly/xSync/pkgs/downloading/mediadownloadhelper"
	"github.com/WangWilly/xSync/pkgs/downloading/resolveworker"
	"github.com/WangWilly/xSync/pkgs/workers"
	"github.com/go-resty/resty/v2"
	"github.com/jmoiron/sqlx"
	log "github.com/sirupsen/logrus"
)

////////////////////////////////////////////////////////////////////////////////

// Global state variables
var (
	MaxDownloadRoutine int
)

// init initializes default configuration values
func init() {
	MaxDownloadRoutine = min(100, runtime.GOMAXPROCS(0)*10)
}

////////////////////////////////////////////////////////////////////////////////

func BatchUserDownload(ctx context.Context, client *resty.Client, db *sqlx.DB, users []heaphelper.UserWithinListEntity, dir string, autoFollow bool, additional []*resty.Client) ([]*dldto.InEntity, error) {
	// uidToUser := make(map[uint64]*twitter.User)
	// for _, u := range users {
	// 	uidToUser[u.User.TwitterId] = u.User
	// }

	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)

	// userSmartPathToDepth := make(map[*smartpathdto.UserSmartPath]int)
	// // 大顶堆，以用户深度
	// userUserSmartPathHeap := utils.NewHeap(func(lhs, rhs *smartpathdto.UserSmartPath) bool {
	// 	luser, ruser := uidToUser[lhs.Uid()], uidToUser[rhs.Uid()]
	// 	lOnlyMater := luser.IsProtected && luser.Followstate == twitter.FS_FOLLOWING
	// 	rOnlyMaster := ruser.IsProtected && ruser.Followstate == twitter.FS_FOLLOWING

	// 	if lOnlyMater == rOnlyMaster {
	// 		return userSmartPathToDepth[lhs] > userSmartPathToDepth[rhs]
	// 	}
	// 	return lOnlyMater // 优先让 master 获取只有他能看到的
	// })

	// // pre-process
	// start := time.Now()
	// deepest := 0
	// missingTweets := 0
	// updaterLogger := log.WithField("worker", "updating")
	// {
	// 	defer utils.PanicHandler(cancel)
	// 	log.Infoln("start pre processing users")

	// 	for _, userWithinList := range users {

	// 		user := userWithinList.User
	// 		if heaphelper.IsIngoreUser(user) {
	// 			continue
	// 		}

	// 		var userSmartPath *smartpathdto.UserSmartPath
	// 		maybeUserSmartPath, loaded := syncedUserSmartPaths.Load(user.TwitterId)
	// 		if !loaded {
	// 			var err error
	// 			userSmartPath, err = heaphelper.SyncUserToDbAndGetSmartPath(db, user, dir)
	// 			if err != nil {
	// 				updaterLogger.WithField("user", user.Title()).Warnln("failed to update user or entity", err)
	// 				continue
	// 			}
	// 			syncedUserSmartPaths.Store(user.TwitterId, userSmartPath)

	// 			// 同步所有现存的指向此用户的符号链接
	// 			linkds, err := database.GetUserLinks(db, user.TwitterId)
	// 			if err != nil {
	// 				updaterLogger.WithField("user", user.Title()).Warnln("failed to get links to user:", err)
	// 			}
	// 			upath, _ := userSmartPath.Path()
	// 			for _, linkd := range linkds {
	// 				if err = heaphelper.UpdateUserLink(linkd, db, upath); err != nil {
	// 					updaterLogger.WithField("user", user.Title()).Warnln("failed to update link:", err)
	// 				}
	// 				sl, _ := syncedListUsers.LoadOrStore(int(linkd.ParentLstEntityId), &sync.Map{})
	// 				syncedList := sl.(*sync.Map)
	// 				syncedList.Store(user.TwitterId, struct{}{})
	// 			}

	// 			// 计算深度
	// 			if user.MediaCount != 0 && user.IsVisiable() {
	// 				missingTweets += max(0, user.MediaCount-int(userSmartPath.Record.MediaCount.Int32))
	// 				userSmartPathToDepth[userSmartPath] = heaphelper.CalcUserDepth(int(userSmartPath.Record.MediaCount.Int32), user.MediaCount)
	// 				userUserSmartPathHeap.Push(userSmartPath)
	// 				deepest = max(deepest, userSmartPathToDepth[userSmartPath])
	// 			}

	// 			// 自动关注
	// 			if user.IsProtected && user.Followstate == twitter.FS_UNFOLLOW && autoFollow {
	// 				if err := twitter.FollowUser(ctx, client, user); err != nil {
	// 					log.WithField("user", user.Title()).Warnln("failed to follow user:", err)
	// 				} else {
	// 					log.WithField("user", user.Title()).Debugln("follow request has been sent")
	// 				}
	// 			}
	// 		} else {
	// 			userSmartPath = maybeUserSmartPath.(*smartpathdto.UserSmartPath)
	// 		}

	// 		// 即便同步一个用户时也同步了所有指向此用户的链接，
	// 		// 但此用户仍可能会是一个新的 “列表-用户”，所以判断此用户链接是否同步过，
	// 		// 如果否，那么创建一个属于此列表的用户链接
	// 		leid := userWithinList.Leid
	// 		if leid == nil {
	// 			continue
	// 		}
	// 		sl, _ := syncedListUsers.LoadOrStore(*leid, &sync.Map{})
	// 		syncedList := sl.(*sync.Map)
	// 		_, loaded = syncedList.LoadOrStore(user.TwitterId, struct{}{})
	// 		if loaded {
	// 			continue
	// 		}

	// 		// 为当前列表的新用户创建符号链接
	// 		upath, _ := userSmartPath.Path()
	// 		var linkname = userSmartPath.Name()

	// 		curlink := &database.UserLink{}
	// 		curlink.Name = linkname
	// 		curlink.ParentLstEntityId = int32(*leid)
	// 		curlink.Uid = user.TwitterId

	// 		linkpath, err := curlink.Path(db)
	// 		if err == nil {
	// 			if err = os.Symlink(upath, linkpath); err == nil || os.IsExist(err) {
	// 				err = database.CreateUserLink(db, curlink)
	// 			}
	// 		}
	// 		if err != nil {
	// 			updaterLogger.WithField("user", user.Title()).Warnln("failed to create link for user:", err)
	// 		}
	// 	}
	// }

	// if userUserSmartPathHeap.Empty() {
	// 	return nil, nil
	// }
	// log.Debugln("preprocessing finish, elapsed:", time.Since(start))
	// log.Debugln("real members:", userUserSmartPathHeap.Size())
	// log.Debugln("missing tweets:", missingTweets)
	// log.Debugln("deepest:", deepest)

	// heapHelper := heaphelper.NewHelperDirect(
	// 	uidToUser,
	// 	userSmartPathToDepth,
	// 	userUserSmartPathHeap,
	// )

	heapHelper := heaphelper.NewHelper(users)
	heapHelper.MakeHeap(ctx, db, client, dir, autoFollow)
	log.Infof("heap size: %d\n", heapHelper.GetHeap().Size())

	mediaDownloadHelper := mediadownloadhelper.NewHelper()
	log.Infoln("start downloading tweets")

	// Create SimpleWorker for tweet media downloading
	// Convert CancelCauseFunc to CancelFunc for SimpleWorker compatibility
	ctxForSimpleWorker, cancelForSimpleWorker := context.WithCancel(ctx)
	simpleWorker := workers.NewSimpleWorker[dldto.TweetDlMeta, dldto.TweetDlMeta](ctxForSimpleWorker, cancelForSimpleWorker, MaxDownloadRoutine)

	// Create the original worker for heap processing and tweet downloading logic
	worker := resolveworker.NewWorker(ctx, cancel, mediaDownloadHelper)

	// Define producer function that processes the heap
	producer := func(ctx context.Context, output chan<- dldto.TweetDlMeta) ([]dldto.TweetDlMeta, error) {
		unsent, err := worker.ProduceFromHeap(heapHelper, db, client, additional, output, simpleWorker.IncrementProduced)
		return unsent, err
	}

	// Define consumer function that downloads tweet media
	consumer := func(ctx context.Context, input <-chan dldto.TweetDlMeta) []dldto.TweetDlMeta {
		return worker.DownloadTweetMediaFromTweetChan(client, input)
	}

	// Run the producer-consumer pipeline
	result := simpleWorker.Process(producer, consumer, MaxDownloadRoutine)

	// Convert failed tweets to InEntity format
	fails := make([]*dldto.InEntity, 0, len(result.Failed))
	for _, failedTweet := range result.Failed {
		if inEntity, ok := failedTweet.(*dldto.InEntity); ok {
			fails = append(fails, inEntity)
		}
	}

	// Log results
	log.WithFields(log.Fields{
		"produced":    result.Stats.Produced,
		"consumed":    result.Stats.Consumed,
		"failed":      result.Stats.Failed,
		"duration":    result.Stats.Duration,
		"failedCount": len(fails),
		"totalCount":  heapHelper.GetHeap().Size(),
	}).Info("[BatchDownloadTweet] finished downloading tweets using SimpleWorker")

	// Check for producer errors
	if result.Error != nil {
		log.WithError(result.Error).Error("Producer error during tweet download")
		return fails, result.Error
	}

	return fails, context.Cause(ctx)
}

////////////////////////////////////////////////////////////////////////////////

// BatchDownloadTweet downloads multiple tweets in parallel and returns failed downloads
// 批量下载推文并返回下载失败的推文，可以保证推文被成功下载或被返回
func BatchDownloadTweet(ctx context.Context, client *resty.Client, tweetDlMetas ...dldto.TweetDlMeta) []dldto.TweetDlMeta {
	if len(tweetDlMetas) == 0 {
		return nil
	}

	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)

	// Create SimpleWorker for tweet downloading
	ctxForSimpleWorker, cancelForSimpleWorker := context.WithCancel(ctx)
	simpleWorker := workers.NewSimpleWorker[dldto.TweetDlMeta, dldto.TweetDlMeta](ctxForSimpleWorker, cancelForSimpleWorker, min(len(tweetDlMetas), MaxDownloadRoutine))

	mediaDownloadHelper := mediadownloadhelper.NewHelper()
	worker := resolveworker.NewWorker(ctx, cancel, mediaDownloadHelper)

	// Define producer function that sends the tweet list
	producer := func(ctx context.Context, output chan<- dldto.TweetDlMeta) ([]dldto.TweetDlMeta, error) {
		var unsent []dldto.TweetDlMeta
		for _, tweetDlMeta := range tweetDlMetas {
			select {
			case <-ctx.Done():
				// Add remaining tweets to unsent list
				for i := len(unsent); i < len(tweetDlMetas); i++ {
					unsent = append(unsent, tweetDlMetas[i])
				}
				return unsent, ctx.Err()
			case output <- tweetDlMeta:
				simpleWorker.IncrementProduced()
			default:
				// Channel is full or blocked, add to unsent
				unsent = append(unsent, tweetDlMeta)
			}
		}
		return unsent, nil
	}

	// Define consumer function that downloads tweet media
	consumer := func(ctx context.Context, input <-chan dldto.TweetDlMeta) []dldto.TweetDlMeta {
		return worker.DownloadTweetMediaFromTweetChan(client, input)
	}

	// Run the producer-consumer pipeline
	result := simpleWorker.Process(producer, consumer, min(len(tweetDlMetas), MaxDownloadRoutine))

	if result.Error != nil {
		log.WithError(result.Error).Error("Producer error during tweet download")
	}

	log.WithFields(log.Fields{
		"produced": result.Stats.Produced,
		"consumed": result.Stats.Consumed,
		"failed":   result.Stats.Failed,
		"duration": result.Stats.Duration,
	}).Info("BatchDownloadTweet completed using SimpleWorker")

	if len(result.Failed) > 0 {
		log.WithField("worker", "downloading").Warnf("failed to download %d tweets", len(result.Failed))
		return result.Failed
	}
	return nil
}
