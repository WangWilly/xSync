package resolveworker

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"syscall"

	"github.com/WangWilly/xSync/pkgs/database"
	"github.com/WangWilly/xSync/pkgs/downloading/dtos/dldto"
	"github.com/WangWilly/xSync/pkgs/downloading/dtos/smartpathdto"
	"github.com/WangWilly/xSync/pkgs/twitter"
	"github.com/WangWilly/xSync/pkgs/utils"
	"github.com/go-resty/resty/v2"
	"github.com/jmoiron/sqlx"
	"github.com/panjf2000/ants/v2"
	log "github.com/sirupsen/logrus"
)

type worker struct {
	ctx    context.Context
	cancel context.CancelCauseFunc

	mediaDownloadHelper MediaDownloadHelper

	userTweetRateLimit     int
	userTweetMaxConcurrent int // avoid DownstreamOverCapacityError

	genCount  int32
	dlCount   int32
	checkCond sync.Cond
}

func NewWorker(ctx context.Context, cancel context.CancelCauseFunc, mediaDownloadHelper MediaDownloadHelper) *worker {
	return &worker{
		ctx:    ctx,
		cancel: cancel,

		mediaDownloadHelper: mediaDownloadHelper,

		userTweetRateLimit:     500, // TODO: make configurable
		userTweetMaxConcurrent: 100, //

		genCount:  0,
		dlCount:   0,
		checkCond: sync.Cond{L: &sync.Mutex{}},
	}
}

////////////////////////////////////////////////////////////////////////////////

func (w *worker) DownloadTweetMediaFromHeap(
	heapHelper HeapHelper,
	db *sqlx.DB,
	client *resty.Client,
	additional []*resty.Client,
	maxDownloadRoutine int,
) error {
	tweetChan := make(chan dldto.TweetDlMeta, maxDownloadRoutine)
	errChan := make(chan dldto.TweetDlMeta)

	return w.DownloadTweetMediaFromHeapWithChan(
		heapHelper,
		db,
		client,
		additional,
		maxDownloadRoutine,
		tweetChan,
		errChan,
	)
}

func (w *worker) DownloadTweetMediaFromHeapWithChan(
	heapHelper HeapHelper,
	db *sqlx.DB,
	client *resty.Client,
	additional []*resty.Client,
	maxDownloadRoutine int,
	tweetChan chan dldto.TweetDlMeta,
	errChan chan dldto.TweetDlMeta,
) error {
	producerPool, err := ants.NewPool(min(w.userTweetMaxConcurrent, heapHelper.GetHeap().Size()))
	if err != nil {
		return err
	}
	defer ants.Release()

	w.pooledSourceFromHeap(producerPool, heapHelper, db, client, additional, tweetChan)

	go func() {
		defer close(tweetChan)
		for {
			w.checkCond.L.Lock()
			defer w.checkCond.L.Unlock()

			if w.genCount == w.dlCount {
				return
			}
		}
	}()

	consumerWg := sync.WaitGroup{}
	for range maxDownloadRoutine {
		consumerWg.Add(1)
		go w.DownloadTweetMediaFromTweetChan(client, errChan, tweetChan)
	}
	consumerWg.Wait()

	return nil
}

////////////////////////////////////////////////////////////////////////////////

func (w *worker) pooledSourceFromHeap(
	producerPool *ants.Pool,
	heapHelper HeapHelper,
	db *sqlx.DB,
	client *resty.Client,
	additional []*resty.Client,
	tweetChan chan<- dldto.TweetDlMeta,
) {
	clients := make([]*resty.Client, 0)
	clients = append(clients, client)
	clients = append(clients, additional...)

	// 按批次调用生产者
	heap := heapHelper.GetHeap()
	prodWg := sync.WaitGroup{}
	for !heap.Empty() && w.ctx.Err() == nil {
		selected := []int{}
		count := 0
		for count < w.userTweetRateLimit && w.ctx.Err() == nil {
			if heap.Empty() {
				break
			}

			entity := heap.Peek()
			depth := heapHelper.GetDepth(entity)
			if depth > w.userTweetRateLimit {
				log.WithFields(log.Fields{
					"user":  entity.Name(),
					"depth": depth,
				}).Warnln("user depth greater than the max limit of window")
				heap.Pop()
				continue
			}

			if count+depth > w.userTweetRateLimit {
				break
			}

			prodWg.Add(1)
			producerPool.Submit(func() {
				defer prodWg.Done()
				w.fetchTweet(entity, heapHelper, db, clients, tweetChan)
			})
			selected = append(selected, depth)

			count += depth
			heap.Pop()
		}
		log.Debugln(selected)
		prodWg.Wait()
	}
}

func (w *worker) fetchTweet(
	entity *smartpathdto.UserSmartPath,
	heapHelper HeapHelper,
	db *sqlx.DB,
	clients []*resty.Client,
	tweetChan chan<- dldto.TweetDlMeta,
) {
	defer utils.PanicHandler(w.cancel)
	logger := log.WithField("worker", "getting")

	user := heapHelper.GetUserByTwitterId(entity.Uid())
	heap := heapHelper.GetHeap()
	if w.ctx.Err() != nil {
		heap.Push(entity)
		return
	}
	cli := twitter.SelectClientForMediaRequest(w.ctx, clients)
	if cli == nil {
		heap.Push(entity)
		w.cancel(fmt.Errorf("no client available"))
		return
	}

	tweets, err := user.GetMeidas(w.ctx, cli, &utils.TimeRange{Min: entity.LatestReleaseTime()})
	if err == twitter.ErrWouldBlock {
		heap.Push(entity)
		return
	}
	if v, ok := err.(*twitter.TwitterApiError); ok {
		// 客户端不再可用
		switch v.Code {
		case twitter.ErrExceedPostLimit:
			twitter.SetClientError(cli, fmt.Errorf("reached the limit for seeing posts today"))
			heap.Push(entity)
			return
		case twitter.ErrAccountLocked:
			twitter.SetClientError(cli, fmt.Errorf("account is locked"))
			heap.Push(entity)
			return
		}
	}
	if w.ctx.Err() != nil {
		heap.Push(entity)
		return
	}
	if err != nil {
		logger.WithField("user", entity.Name()).Warnln("failed to get user medias:", err)
		return
	}

	if len(tweets) == 0 {
		if err := database.UpdateUserEntityMediCount(db, entity.Id(), user.MediaCount); err != nil {
			logger.WithField("user", entity.Name()).Panicln("failed to update user medias count:", err)
		}
		return
	}

	// 确保该用户所有推文已推送并更新用户推文状态
	for _, tw := range tweets {
		pt := dldto.InEntity{Tweet: tw, Entity: entity}
		select {
		case tweetChan <- &pt:
			atomic.AddInt32(&w.genCount, 1)
		case <-w.ctx.Done():
			return // 防止无消费者导致死锁
		}
	}

	if err := database.UpdateUserEntityTweetStat(db, entity.Id(), tweets[0].CreatedAt, user.MediaCount); err != nil {
		// 影响程序的正确性，必须 Panic
		logger.WithField("user", entity.Name()).Panicln("failed to update user tweets stat:", err)
	}
}

////////////////////////////////////////////////////////////////////////////////

// tweetDownloader handles downloading of tweets from a channel
// 负责下载推文，保证 tweet chan 内的推文要么下载成功，要么推送至 error chan
func (w *worker) DownloadTweetMediaFromTweetChan(client *resty.Client, errChan chan<- dldto.TweetDlMeta, tweetChan <-chan dldto.TweetDlMeta) {
	var pt dldto.TweetDlMeta
	var ok bool

	defer w.checkCond.Signal()
	defer func() {
		if p := recover(); p != nil {
			w.cancel(fmt.Errorf("%v", p)) // panic 取消上下文，防止生产者死锁
			log.WithField("worker", "downloading").Errorln("panic:", p)

			dlCount := 0
			if pt != nil {
				errChan <- pt // push 正下载的推文
				dlCount += 1
			}
			// 确保只有1个协程的情况下，未能下载完毕的推文仍然会全部推送到 errch
			for pt := range tweetChan {
				errChan <- pt
				dlCount += 1
			}

			atomic.AddInt32(&w.dlCount, int32(dlCount))
		}
	}()

	for {
		select {
		case pt, ok = <-tweetChan:
			if !ok {
				return
			}
		case <-w.ctx.Done():
			dlCount := 0
			for pt := range tweetChan {
				errChan <- pt
				dlCount += 1
			}
			atomic.AddInt32(&w.dlCount, int32(dlCount))
			return
		}

		err := w.mediaDownloadHelper.SafeDownload(w.ctx, client, pt)
		atomic.AddInt32(&w.dlCount, 1)
		if err == nil {
			continue
		}

		errChan <- pt
		// cancel context and exit if no disk space
		if errors.Is(err, syscall.ENOSPC) {
			w.cancel(err)
		}
	}
}
