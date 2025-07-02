package downloading

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/WangWilly/xSync/pkgs/database"
	"github.com/WangWilly/xSync/pkgs/downloading/dtos/packedtweetdto"
	"github.com/WangWilly/xSync/pkgs/downloading/dtos/smartpathdto"
	"github.com/WangWilly/xSync/pkgs/twitter"
	"github.com/WangWilly/xSync/pkgs/utils"
	"github.com/go-resty/resty/v2"
	"github.com/gookit/color"
	"github.com/jmoiron/sqlx"
	"github.com/panjf2000/ants/v2"
	log "github.com/sirupsen/logrus"
)

////////////////////////////////////////////////////////////////////////////////

// Download configuration constants
const (
	userTweetRateLimit     = 500
	userTweetMaxConcurrent = 100 // avoid DownstreamOverCapacityError
)

// Global state variables
var (
	mutex                sync.Mutex
	MaxDownloadRoutine   int
	syncedUserSmartPaths sync.Map // map[user_id]*UserEntity - tracks synced users for current run
	syncedListUsers      sync.Map // leid -> uid -> struct{} - tracks synced list users
)

// workerConfig holds configuration for download workers
type workerConfig struct {
	ctx    context.Context
	wg     *sync.WaitGroup
	cancel context.CancelCauseFunc
}

// init initializes default configuration values
func init() {
	MaxDownloadRoutine = min(100, runtime.GOMAXPROCS(0)*10)
}

////////////////////////////////////////////////////////////////////////////////

func BatchUserDownload(ctx context.Context, client *resty.Client, db *sqlx.DB, users []userWithinListEntity, dir string, autoFollow bool, additional []*resty.Client) ([]*packedtweetdto.InEntity, error) {
	uidToUser := make(map[uint64]*twitter.User)
	for _, u := range users {
		uidToUser[u.user.TwitterId] = u.user
	}

	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)

	depthByEntity := make(map[*smartpathdto.UserSmartPath]int)
	// 大顶堆，以用户深度
	userEntityHeap := utils.NewHeap(func(lhs, rhs *smartpathdto.UserSmartPath) bool {
		luser, ruser := uidToUser[lhs.Uid()], uidToUser[rhs.Uid()]
		lOnlyMater := luser.IsProtected && luser.Followstate == twitter.FS_FOLLOWING
		rOnlyMaster := ruser.IsProtected && ruser.Followstate == twitter.FS_FOLLOWING

		if lOnlyMater == rOnlyMaster {
			return depthByEntity[lhs] > depthByEntity[rhs]
		}
		return lOnlyMater // 优先让 master 获取只有他能看到的
	})

	start := time.Now()
	deepest := 0

	// pre-process
	missingTweets := 0
	updaterLogger := log.WithField("worker", "updating")
	{
		defer utils.PanicHandler(cancel)
		log.Infoln("start pre processing users")

		for _, userWithinList := range users {

			user := userWithinList.user
			if shouldIngoreUser(user) {
				continue
			}

			var userSmartPath *smartpathdto.UserSmartPath
			maybeUserSmartPath, loaded := syncedUserSmartPaths.Load(user.TwitterId)
			if !loaded {
				var err error
				userSmartPath, err = syncUserToDbAndGetSmartPath(db, user, dir)
				if err != nil {
					updaterLogger.WithField("user", user.Title()).Warnln("failed to update user or entity", err)
					continue
				}
				syncedUserSmartPaths.Store(user.TwitterId, userSmartPath)

				// 同步所有现存的指向此用户的符号链接
				linkds, err := database.GetUserLinks(db, user.TwitterId)
				if err != nil {
					updaterLogger.WithField("user", user.Title()).Warnln("failed to get links to user:", err)
				}
				upath, _ := userSmartPath.Path()
				for _, linkd := range linkds {
					if err = updateUserLink(linkd, db, upath); err != nil {
						updaterLogger.WithField("user", user.Title()).Warnln("failed to update link:", err)
					}
					sl, _ := syncedListUsers.LoadOrStore(int(linkd.ParentLstEntityId), &sync.Map{})
					syncedList := sl.(*sync.Map)
					syncedList.Store(user.TwitterId, struct{}{})
				}

				// 计算深度
				if user.MediaCount != 0 && user.IsVisiable() {
					missingTweets += max(0, user.MediaCount-int(userSmartPath.Record.MediaCount.Int32))
					depthByEntity[userSmartPath] = calcUserDepth(int(userSmartPath.Record.MediaCount.Int32), user.MediaCount)
					userEntityHeap.Push(userSmartPath)
					deepest = max(deepest, depthByEntity[userSmartPath])
				}

				// 自动关注
				if user.IsProtected && user.Followstate == twitter.FS_UNFOLLOW && autoFollow {
					if err := twitter.FollowUser(ctx, client, user); err != nil {
						log.WithField("user", user.Title()).Warnln("failed to follow user:", err)
					} else {
						log.WithField("user", user.Title()).Debugln("follow request has been sent")
					}
				}
			} else {
				userSmartPath = maybeUserSmartPath.(*smartpathdto.UserSmartPath)
			}

			// 即便同步一个用户时也同步了所有指向此用户的链接，
			// 但此用户仍可能会是一个新的 “列表-用户”，所以判断此用户链接是否同步过，
			// 如果否，那么创建一个属于此列表的用户链接
			leid := userWithinList.leid
			if leid == nil {
				continue
			}
			sl, _ := syncedListUsers.LoadOrStore(*leid, &sync.Map{})
			syncedList := sl.(*sync.Map)
			_, loaded = syncedList.LoadOrStore(user.TwitterId, struct{}{})
			if loaded {
				continue
			}

			// 为当前列表的新用户创建符号链接
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
				updaterLogger.WithField("user", user.Title()).Warnln("failed to create link for user:", err)
			}
		}
	}

	if userEntityHeap.Empty() {
		return nil, nil
	}
	log.Debugln("preprocessing finish, elapsed:", time.Since(start))
	log.Debugln("real members:", userEntityHeap.Size())
	log.Debugln("missing tweets:", missingTweets)
	log.Debugln("deepest:", deepest)

	clients := make([]*resty.Client, 0)
	clients = append(clients, client)
	clients = append(clients, additional...)

	getterLogger := log.WithField("worker", "getting")
	prodwg := sync.WaitGroup{}
	tweetChan := make(chan packedtweetdto.PackedTweet, MaxDownloadRoutine)
	errChan := make(chan packedtweetdto.PackedTweet)
	producer := func(entity *smartpathdto.UserSmartPath) {
		defer prodwg.Done()
		defer utils.PanicHandler(cancel)

		user := uidToUser[entity.Uid()]
		cli := twitter.SelectUserMediaClient(ctx, clients)
		if ctx.Err() != nil {
			userEntityHeap.Push(entity)
			return
		}
		if cli == nil {
			userEntityHeap.Push(entity)
			cancel(fmt.Errorf("no client available"))
			return
		}

		tweets, err := user.GetMeidas(ctx, cli, &utils.TimeRange{Min: entity.LatestReleaseTime()})
		if err == twitter.ErrWouldBlock {
			userEntityHeap.Push(entity)
			return
		}
		if v, ok := err.(*twitter.TwitterApiError); ok {
			// 客户端不再可用
			switch v.Code {
			case twitter.ErrExceedPostLimit:
				twitter.SetClientError(cli, fmt.Errorf("reached the limit for seeing posts today"))
				userEntityHeap.Push(entity)
				return
			case twitter.ErrAccountLocked:
				twitter.SetClientError(cli, fmt.Errorf("account is locked"))
				userEntityHeap.Push(entity)
				return
			}
		}
		if ctx.Err() != nil {
			userEntityHeap.Push(entity)
			return
		}
		if err != nil {
			getterLogger.WithField("user", entity.Name()).Warnln("failed to get user medias:", err)
			return
		}

		if len(tweets) == 0 {
			if err := database.UpdateUserEntityMediCount(db, entity.Id(), user.MediaCount); err != nil {
				getterLogger.WithField("user", entity.Name()).Panicln("failed to update user medias count:", err)
			}
			return
		}

		// 确保该用户所有推文已推送并更新用户推文状态
		for _, tw := range tweets {
			pt := packedtweetdto.InEntity{Tweet: tw, Entity: entity}
			select {
			case tweetChan <- &pt:
			case <-ctx.Done():
				return // 防止无消费者导致死锁
			}
		}

		if err := database.UpdateUserEntityTweetStat(db, entity.Id(), tweets[0].CreatedAt, user.MediaCount); err != nil {
			// 影响程序的正确性，必须 Panic
			getterLogger.WithField("user", entity.Name()).Panicln("failed to update user tweets stat:", err)
		}
	}

	// launch worker
	conswg := sync.WaitGroup{}
	config := workerConfig{
		ctx:    ctx,
		wg:     &conswg,
		cancel: cancel,
	}
	for i := 0; i < MaxDownloadRoutine; i++ {
		conswg.Add(1)
		go tweetDownloader(client, &config, errChan, tweetChan)
	}

	producerPool, err := ants.NewPool(min(userTweetMaxConcurrent, userEntityHeap.Size()))
	if err != nil {
		return nil, err
	}
	defer ants.Release()

	//closer
	go func() {
		// 按批次调用生产者
		for !userEntityHeap.Empty() && ctx.Err() == nil {
			selected := []int{}
			for count := 0; count < userTweetRateLimit && ctx.Err() == nil; {
				if userEntityHeap.Empty() {
					break
				}

				entity := userEntityHeap.Peek()
				depth := depthByEntity[entity]
				if depth > userTweetRateLimit {
					log.WithFields(log.Fields{
						"user":  entity.Name(),
						"depth": depth,
					}).Warnln("user depth greater than the max limit of window")
					userEntityHeap.Pop()
					continue
				}

				if depth+count > userTweetRateLimit {
					break
				}

				prodwg.Add(1)
				producerPool.Submit(func() {
					producer(entity)
				})
				selected = append(selected, depth)

				count += depth
				//delete(depthByEntity, entity)
				userEntityHeap.Pop()
			}
			log.Debugln(selected)
			prodwg.Wait()
		}
		close(tweetChan)
		log.Debugf("getting tweets completed, elapsed time: %v", time.Since(start))

		conswg.Wait()
		close(errChan)
	}()

	fails := []*packedtweetdto.InEntity{}
	for pt := range errChan {
		fails = append(fails, pt.(*packedtweetdto.InEntity))
	}
	log.Debugf("%d users unable to start", userEntityHeap.Size())
	return fails, context.Cause(ctx)
}

////////////////////////////////////////////////////////////////////////////////

// BatchDownloadTweet downloads multiple tweets in parallel and returns failed downloads
// 批量下载推文并返回下载失败的推文，可以保证推文被成功下载或被返回
func BatchDownloadTweet(ctx context.Context, client *resty.Client, pts ...packedtweetdto.PackedTweet) []packedtweetdto.PackedTweet {
	if len(pts) == 0 {
		return nil
	}

	ctx, cancel := context.WithCancelCause(ctx)

	var errChan = make(chan packedtweetdto.PackedTweet)
	var tweetChan = make(chan packedtweetdto.PackedTweet, len(pts))
	var wg sync.WaitGroup // number of working goroutines
	var numRoutine = min(len(pts), MaxDownloadRoutine)

	for _, pt := range pts {
		tweetChan <- pt
	}
	close(tweetChan)

	config := workerConfig{
		ctx:    ctx,
		cancel: cancel,
		wg:     &wg,
	}
	for range numRoutine {
		wg.Add(1)
		go tweetDownloader(client, &config, errChan, tweetChan)
	}

	go func() {
		wg.Wait()
		close(errChan)
	}()

	errors := []packedtweetdto.PackedTweet{}
	for pt := range errChan {
		errors = append(errors, pt)
	}
	return errors
}

////////////////////////////////////////////////////////////////////////////////
// Download Worker Operations
////////////////////////////////////////////////////////////////////////////////

// tweetDownloader handles downloading of tweets from a channel
// 负责下载推文，保证 tweet chan 内的推文要么下载成功，要么推送至 error chan
func tweetDownloader(client *resty.Client, config *workerConfig, errch chan<- packedtweetdto.PackedTweet, twech <-chan packedtweetdto.PackedTweet) {
	var pt packedtweetdto.PackedTweet
	var ok bool

	defer config.wg.Done()
	defer func() {
		if p := recover(); p != nil {
			config.cancel(fmt.Errorf("%v", p)) // panic 取消上下文，防止生产者死锁
			log.WithField("worker", "downloading").Errorln("panic:", p)

			if pt != nil {
				errch <- pt // push 正下载的推文
			}
			// 确保只有1个协程的情况下，未能下载完毕的推文仍然会全部推送到 errch
			for pt := range twech {
				errch <- pt
			}
		}
	}()

	for {
		select {
		case pt, ok = <-twech:
			if !ok {
				return
			}
		case <-config.ctx.Done():
			for pt := range twech {
				errch <- pt
			}
			return
		}

		path := pt.GetPath()
		if path == "" {
			errch <- pt
			continue
		}
		err := downloadTweetMedia(config.ctx, client, path, pt.GetTweet())
		// 403: Dmcaed
		if err != nil && !utils.IsStatusCode(err, 404) && !utils.IsStatusCode(err, 403) {
			errch <- pt
		}

		// cancel context and exit if no disk space
		if errors.Is(err, syscall.ENOSPC) {
			config.cancel(err)
		}
	}
}

////////////////////////////////////////////////////////////////////////////////
// Tweet Media Download Operations
////////////////////////////////////////////////////////////////////////////////

// downloadTweetMedia downloads all media files for a given tweet
// 任何一个 url 下载失败直接返回
// TODO: 要么全做，要么不做
func downloadTweetMedia(ctx context.Context, client *resty.Client, dir string, tweet *twitter.Tweet) error {
	text := utils.ToLegalWindowsFileName(tweet.Text)

	for _, u := range tweet.Urls {
		ext, err := utils.GetExtFromUrl(u)
		if err != nil {
			return err
		}

		// 请求
		resp, err := client.R().SetContext(ctx).SetQueryParam("name", "4096x4096").Get(u)
		if err != nil {
			return err
		}

		mutex.Lock()
		path, err := utils.UniquePath(filepath.Join(dir, text+ext))
		if err != nil {
			mutex.Unlock()
			return err
		}
		file, err := os.Create(path)
		mutex.Unlock()
		if err != nil {
			return err
		}

		defer os.Chtimes(path, time.Time{}, tweet.CreatedAt)
		defer file.Close()

		_, err = file.Write(resp.Body())
		if err != nil {
			return err
		}
	}

	fmt.Printf("%s %s\n", color.FgLightMagenta.Render("["+tweet.Creator.Title()+"]"), text)
	return nil
}
