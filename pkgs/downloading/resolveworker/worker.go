package resolveworker

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

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

	genCount int32
	dlCount  int32
}

func NewWorker(ctx context.Context, cancel context.CancelCauseFunc, mediaDownloadHelper MediaDownloadHelper) *worker {
	return &worker{
		ctx:    ctx,
		cancel: cancel,

		mediaDownloadHelper: mediaDownloadHelper,

		userTweetRateLimit:     500, // TODO: make configurable
		userTweetMaxConcurrent: 100, //

		genCount: 0,
		dlCount:  0,
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
	defer func() {
		log.WithField("worker", "coordinator").Info("finished")
	}()

	// Use channels for better producer-consumer coordination
	producerDone := make(chan struct{})
	producerErr := make(chan error, 1)

	// Start producer in a separate goroutine
	go func() {
		defer close(producerDone)
		if err := w.runProducers(producerPool, heapHelper, db, client, additional, tweetChan); err != nil {
			select {
			case producerErr <- err:
			default:
			}
		}
	}()

	// Start channel closer goroutine
	go func() {
		defer close(tweetChan)
		select {
		case <-producerDone:
			log.WithField("worker", "coordinator").Info("all producers finished, closing tweet channel")
		case <-w.ctx.Done():
			log.WithField("worker", "coordinator").Info("context cancelled, closing tweet channel")
		}
	}()

	// Start consumers
	consumerWg := sync.WaitGroup{}
	var allFailedTweets [][]dldto.TweetDlMeta
	var failedTweetsMutex sync.Mutex

	for i := 0; i < maxDownloadRoutine; i++ {
		consumerWg.Add(1)
		go func(consumerID int) {
			defer consumerWg.Done()
			logger := log.WithField("consumerID", consumerID)
			logger.Info("starting consumer")

			failedTweets := w.DownloadTweetMediaFromTweetChan(client, tweetChan)

			// Safely append failed tweets to the shared slice
			failedTweetsMutex.Lock()
			allFailedTweets = append(allFailedTweets, failedTweets)
			failedTweetsMutex.Unlock()

			logger.WithField("failedCount", len(failedTweets)).Info("consumer finished")
		}(i)
	}

	// Wait for all consumers to finish
	consumerWg.Wait()

	// After all consumers finish, handle failed tweets and remaining tweets
	log.WithField("worker", "coordinator").Debug("all consumers finished, handling remaining tweets")

	// Check if there are any remaining tweets that weren't processed
	remainingTweets := []dldto.TweetDlMeta{}

	// Drain any remaining tweets from tweetChan (shouldn't happen normally but safety check)
	for {
		select {
		case pt, ok := <-tweetChan:
			if !ok {
				// Channel is closed, no more tweets
				break
			}
			remainingTweets = append(remainingTweets, pt)
			atomic.AddInt32(&w.dlCount, 1)
		default:
			// No more tweets available immediately
			goto handleRemaining
		}
	}

handleRemaining:
	// Collect all failed tweets from consumers
	var totalFailedTweets []dldto.TweetDlMeta
	for _, consumerFailedTweets := range allFailedTweets {
		totalFailedTweets = append(totalFailedTweets, consumerFailedTweets...)
	}

	// Add any remaining tweets
	totalFailedTweets = append(totalFailedTweets, remainingTweets...)

	// Push all failed tweets to error channel
	for _, pt := range totalFailedTweets {
		select {
		case errChan <- pt:
			log.WithField("tweet", pt.GetTweet().Id).Debug("pushed failed tweet to error channel")
		default:
			log.WithField("tweet", pt.GetTweet().Id).Warn("error channel full, dropping failed tweet")
		}
	}

	// Log final counts for debugging
	finalGenCount := atomic.LoadInt32(&w.genCount)
	finalDlCount := atomic.LoadInt32(&w.dlCount)

	// Validate that all generated tweets have been processed
	if finalGenCount != finalDlCount {
		log.WithFields(log.Fields{
			"genCount":  finalGenCount,
			"dlCount":   finalDlCount,
			"remaining": len(remainingTweets),
			"failed":    len(totalFailedTweets),
			"missing":   finalGenCount - finalDlCount,
		}).Warn("count mismatch detected - some tweets may not have been processed")
	}

	log.WithFields(log.Fields{
		"genCount":  finalGenCount,
		"dlCount":   finalDlCount,
		"remaining": len(remainingTweets),
		"failed":    len(totalFailedTweets),
	}).Info("closing error channel")

	close(errChan)

	// Check for producer errors
	select {
	case err := <-producerErr:
		return fmt.Errorf("producer error: %w", err)
	default:
	}

	return nil
}

func (w *worker) DownloadTweetMediaFromList(
	tweetDlMetas []dldto.TweetDlMeta,
	client *resty.Client,
	maxDownloadRoutine int,
) ([]dldto.TweetDlMeta, error) {
	if len(tweetDlMetas) == 0 {
		return nil, nil
	}

	tweetChan := make(chan dldto.TweetDlMeta, maxDownloadRoutine)

	// Start producer goroutine
	go func() {
		defer close(tweetChan)
		for _, pt := range tweetDlMetas {
			select {
			case tweetChan <- pt:
				atomic.AddInt32(&w.genCount, 1)
			case <-w.ctx.Done():
				return
			}
		}
	}()

	// Start consumers
	consumerWg := sync.WaitGroup{}
	var allFailedTweets [][]dldto.TweetDlMeta
	var failedTweetsMutex sync.Mutex

	for i := 0; i < maxDownloadRoutine; i++ {
		consumerWg.Add(1)
		go func(consumerID int) {
			defer consumerWg.Done()
			logger := log.WithField("consumerID", consumerID)
			logger.Debug("starting list consumer")

			failedTweets := w.DownloadTweetMediaFromTweetChan(client, tweetChan)

			// Safely append failed tweets to the shared slice
			failedTweetsMutex.Lock()
			allFailedTweets = append(allFailedTweets, failedTweets)
			failedTweetsMutex.Unlock()

			logger.WithField("failedCount", len(failedTweets)).Debug("list consumer finished")
		}(i)
	}

	// Wait for all consumers to finish
	consumerWg.Wait()

	// Before handling results, ensure all unfetched tweets are handled
	log.WithField("worker", "list").Debug("all consumers finished, handling remaining tweets")

	// Check if there are any remaining tweets that weren't processed
	remainingTweets := []dldto.TweetDlMeta{}

	// Drain any remaining tweets from tweetChan (shouldn't happen normally but safety check)
	for {
		select {
		case pt, ok := <-tweetChan:
			if !ok {
				// Channel is closed, no more tweets
				break
			}
			remainingTweets = append(remainingTweets, pt)
			atomic.AddInt32(&w.dlCount, 1)
		default:
			// No more tweets available immediately
			goto handleRemainingList
		}
	}

handleRemainingList:
	// Collect all failed tweets from consumers
	var totalFailedTweets []dldto.TweetDlMeta
	for _, consumerFailedTweets := range allFailedTweets {
		totalFailedTweets = append(totalFailedTweets, consumerFailedTweets...)
	}

	// Add any remaining tweets
	totalFailedTweets = append(totalFailedTweets, remainingTweets...)

	// Log final counts for debugging
	finalGenCount := atomic.LoadInt32(&w.genCount)
	finalDlCount := atomic.LoadInt32(&w.dlCount)

	// Validate that all generated tweets have been processed
	if finalGenCount != finalDlCount {
		log.WithFields(log.Fields{
			"genCount":  finalGenCount,
			"dlCount":   finalDlCount,
			"remaining": len(remainingTweets),
			"failed":    len(totalFailedTweets),
			"missing":   finalGenCount - finalDlCount,
		}).Warn("list processing count mismatch detected - some tweets may not have been processed")
	}

	log.WithFields(log.Fields{
		"genCount":  finalGenCount,
		"dlCount":   finalDlCount,
		"remaining": len(remainingTweets),
		"failed":    len(totalFailedTweets),
	}).Info("list processing completed")

	if len(totalFailedTweets) > 0 {
		log.WithField("worker", "downloading").Warnf("failed to download %d tweets", len(totalFailedTweets))
		return totalFailedTweets, fmt.Errorf("failed to download %d tweets", len(totalFailedTweets))
	}

	return nil, nil
}

////////////////////////////////////////////////////////////////////////////////

func (w *worker) fetchTweetOrFailbackToHeap(
	entity *smartpathdto.UserSmartPath,
	heapHelper HeapHelper,
	db *sqlx.DB,
	clients []*resty.Client,
	tweetChan chan<- dldto.TweetDlMeta,
) {
	defer utils.PanicHandler(w.cancel)
	logger := log.WithField("function", "fetchTweetOrFailbackToHeap")

	logger.WithField("user", entity.Name()).Infoln("fetching user tweets")
	user := heapHelper.GetUserByTwitterId(entity.Uid())
	heap := heapHelper.GetHeap()

	// Helper function to safely push back to heap
	safePushToHeap := func(reason string) {
		logger.WithField("user", entity.Name()).Warnf("%s, pushing back to heap", reason)
		// Use a goroutine to avoid blocking on heap operations
		go func() {
			defer func() {
				if r := recover(); r != nil {
					logger.WithField("user", entity.Name()).Errorf("panic while pushing to heap: %v", r)
				}
			}()
			heap.Push(entity)
		}()
	}

	if w.ctx.Err() != nil {
		safePushToHeap("context cancelled")
		return
	}

	logger.WithField("user", entity.Name()).Infof("latest release time: %s", entity.LatestReleaseTime())
	cli := twitter.SelectClientForMediaRequest(w.ctx, clients)
	if cli == nil {
		safePushToHeap("no client available")
		w.cancel(fmt.Errorf("no client available"))
		return
	}

	logger.WithField("user", entity.Name()).Infoln("getting user medias")
	tweets, err := user.GetMeidas(w.ctx, cli, &utils.TimeRange{Min: entity.LatestReleaseTime()})
	if err == twitter.ErrWouldBlock {
		safePushToHeap("client would block")
		return
	}

	logger.WithField("user", entity.Name()).Infoln("got user medias")
	if v, ok := err.(*twitter.TwitterApiError); ok {
		logger.WithField("user", entity.Name()).Warnf("twitter api error: %s", v.Error())
		// 客户端不再可用
		switch v.Code {
		case twitter.ErrExceedPostLimit:
			twitter.SetClientError(cli, fmt.Errorf("reached the limit for seeing posts today"))
			safePushToHeap("exceed post limit")
			return
		case twitter.ErrAccountLocked:
			twitter.SetClientError(cli, fmt.Errorf("account is locked"))
			safePushToHeap("account locked")
			return
		}
	}

	if w.ctx.Err() != nil {
		safePushToHeap("context cancelled while getting user medias")
		return
	}

	if err != nil {
		logger.WithField("user", entity.Name()).Warnln("failed to get user medias:", err)
		return
	}

	if len(tweets) == 0 {
		logger.WithField("user", entity.Name()).Infoln("no tweets found, updating user medias count")
		if err := database.UpdateUserEntityMediCount(db, entity.Id(), user.MediaCount); err != nil {
			logger.WithField("user", entity.Name()).Panicln("failed to update user medias count:", err)
		}
		return
	}

	logger.WithFields(log.Fields{
		"user":     entity.Name(),
		"tweetNum": len(tweets),
	}).Infoln("found tweets, preparing to push to tweet channel")

	// 确保该用户所有推文已推送并更新用户推文状态
	// Use a timeout to prevent indefinite blocking
	pushTimeout := 120 * time.Second
	for _, tw := range tweets {
		pt := dldto.InEntity{Tweet: tw, Entity: entity}

		// Create a fresh timeout for each tweet push
		timeoutTimer := time.NewTimer(pushTimeout)

		select {
		case tweetChan <- &pt:
			timeoutTimer.Stop() // Stop the timer to prevent resource leak
			logger.WithField("user", entity.Name()).Infof("pushed tweet %d to tweet channel", tw.Id)
			atomic.AddInt32(&w.genCount, 1)
			logger.WithField("user", entity.Name()).Infof("genCount: %d", atomic.LoadInt32(&w.genCount))
		case <-w.ctx.Done():
			timeoutTimer.Stop() // Stop the timer to prevent resource leak
			logger.WithField("user", entity.Name()).Warnln("context cancelled while pushing tweets")
			return // 防止无消费者导致死锁
		case <-timeoutTimer.C:
			logger.WithField("user", entity.Name()).Warnln("timeout while pushing tweet to channel")
			return
		}
	}

	logger.WithField("user", entity.Name()).Infoln("updating user medias count in database")
	if err := database.UpdateUserEntityTweetStat(db, entity.Id(), tweets[0].CreatedAt, user.MediaCount); err != nil {
		// 影响程序的正确性，必须 Panic
		logger.WithField("user", entity.Name()).Panicln("failed to update user tweets stat:", err)
	}
}

////////////////////////////////////////////////////////////////////////////////

// DownloadTweetMediaFromTweetChan handles downloading of tweets from a channel
// Returns a slice of tweets that failed to download or were not processed
func (w *worker) DownloadTweetMediaFromTweetChan(client *resty.Client, tweetChan <-chan dldto.TweetDlMeta) []dldto.TweetDlMeta {
	logger := log.WithField("function", "DownloadTweetMediaFromTweetChan")
	logger.Debug("consumer started")

	var failedTweets []dldto.TweetDlMeta

	defer func() {
		logger.WithField("failedCount", len(failedTweets)).Debug("consumer finished")
	}()

	defer func() {
		if p := recover(); p != nil {
			logger.Errorf("consumer panic: %v", p)
			w.cancel(fmt.Errorf("consumer panic: %v", p))

			// Drain remaining tweets and add them to failed list
			drainedCount := 0
			for pt := range tweetChan {
				atomic.AddInt32(&w.dlCount, 1)
				drainedCount++
				failedTweets = append(failedTweets, pt)
				logger.WithField("tweet", pt.GetTweet().Id).Debug("added panic-drained tweet to failed list")
			}
			logger.WithField("drainedCount", drainedCount).Debug("finished draining tweets due to panic")
		}
	}()

	for {
		select {
		case pt, ok := <-tweetChan:
			if !ok {
				logger.Debug("tweet channel closed, consumer exiting")
				return failedTweets
			}

			logger.WithField("tweet", pt.GetTweet().Id).Debug("processing tweet")

			// Download the tweet media
			err := w.mediaDownloadHelper.SafeDownload(w.ctx, client, pt)
			atomic.AddInt32(&w.dlCount, 1)

			if err != nil {
				logger.WithField("tweet", pt.GetTweet().Id).Errorf("failed to download tweet: %v", err)
				failedTweets = append(failedTweets, pt)

				// Cancel context and exit if critical errors occur
				if errors.Is(err, syscall.ENOSPC) {
					logger.Error("no disk space, cancelling context")
					w.cancel(err)
				}
				return failedTweets
			} else {
				logger.WithField("tweet", pt.GetTweet().Id).Debug("downloaded tweet successfully")
			}

		case <-w.ctx.Done():
			logger.Debug("context cancelled, draining remaining tweets")
			// Drain remaining tweets and add them to failed list
			drainedCount := 0
			for pt := range tweetChan {
				atomic.AddInt32(&w.dlCount, 1)
				drainedCount++
				failedTweets = append(failedTweets, pt)
				logger.WithField("tweet", pt.GetTweet().Id).Debug("added drained tweet to failed list")
			}
			logger.WithField("drainedCount", drainedCount).Debug("finished draining tweets due to context cancellation")
			return failedTweets
		}
	}
}

// runProducers coordinates the producer pool and heap processing
func (w *worker) runProducers(
	producerPool *ants.Pool,
	heapHelper HeapHelper,
	db *sqlx.DB,
	client *resty.Client,
	additional []*resty.Client,
	tweetChan chan<- dldto.TweetDlMeta,
) error {
	clients := make([]*resty.Client, 0)
	clients = append(clients, client)
	clients = append(clients, additional...)

	log.WithField("worker", "producer").Info("starting producer pool")
	heap := heapHelper.GetHeap()
	log.WithField("worker", "producer").Infof("initial heap size: %d", heap.Size())

	for !heap.Empty() && w.ctx.Err() == nil {
		log.WithField("worker", "producer").Debug("processing batch from heap")

		if err := w.processBatchFromHeap(producerPool, heapHelper, db, clients, tweetChan); err != nil {
			return fmt.Errorf("failed to process batch: %w", err)
		}
	}

	if w.ctx.Err() != nil {
		return w.ctx.Err()
	}

	log.WithField("worker", "producer").Info("all producers finished successfully")
	return nil
}

// processBatchFromHeap processes a batch of users from the heap
func (w *worker) processBatchFromHeap(
	producerPool *ants.Pool,
	heapHelper HeapHelper,
	db *sqlx.DB,
	clients []*resty.Client,
	tweetChan chan<- dldto.TweetDlMeta,
) error {
	heap := heapHelper.GetHeap()
	prodWg := sync.WaitGroup{}
	batchSize := 0

	for batchSize < w.userTweetRateLimit && w.ctx.Err() == nil {
		if heap.Empty() {
			break
		}

		entity := heap.Peek()
		depth := heapHelper.GetDepth(entity)

		if depth > w.userTweetRateLimit {
			log.WithFields(log.Fields{
				"user":  entity.Name(),
				"depth": depth,
			}).Warn("user depth exceeds rate limit, skipping")
			heap.Pop()
			continue
		}

		if batchSize+depth > w.userTweetRateLimit {
			break
		}

		log.WithFields(log.Fields{
			"user":  entity.Name(),
			"depth": depth,
		}).Debug("submitting user to producer pool")

		prodWg.Add(1)
		err := producerPool.Submit(func() {
			defer prodWg.Done()
			w.fetchTweetOrFailbackToHeap(entity, heapHelper, db, clients, tweetChan)
		})

		if err != nil {
			prodWg.Done()
			return fmt.Errorf("failed to submit task to producer pool: %w", err)
		}

		batchSize += depth
		heap.Pop()
	}

	prodWg.Wait()
	return nil
}

////////////////////////////////////////////////////////////////////////////////

// ProduceFromHeap produces tweets from heap for SimpleWorker pattern
func (w *worker) ProduceFromHeap(
	heapHelper HeapHelper,
	db *sqlx.DB,
	client *resty.Client,
	additional []*resty.Client,
	output chan<- dldto.TweetDlMeta,
	incrementProduced func(),
) ([]dldto.TweetDlMeta, error) {
	var unsentTweets []dldto.TweetDlMeta

	clients := make([]*resty.Client, 0)
	clients = append(clients, client)
	clients = append(clients, additional...)

	log.WithField("worker", "producer").Info("starting simple producer for SimpleWorker")
	heap := heapHelper.GetHeap()
	log.WithField("worker", "producer").Infof("initial heap size: %d", heap.Size())

	// Process users from heap sequentially
	for !heap.Empty() && w.ctx.Err() == nil {
		entity := heap.Peek()
		heap.Pop()

		log.WithField("user", entity.Name()).Debug("processing user from heap")

		w.fetchTweetOrFailbackToHeapForSimpleWorker(entity, heapHelper, db, clients, output, incrementProduced)
	}

	if w.ctx.Err() != nil {
		return unsentTweets, w.ctx.Err()
	}

	log.WithField("worker", "producer").Info("all producers finished successfully for SimpleWorker")
	return unsentTweets, nil
}

// fetchTweetOrFailbackToHeapForSimpleWorker is adapted for SimpleWorker pattern
func (w *worker) fetchTweetOrFailbackToHeapForSimpleWorker(
	entity *smartpathdto.UserSmartPath,
	heapHelper HeapHelper,
	db *sqlx.DB,
	clients []*resty.Client,
	output chan<- dldto.TweetDlMeta,
	incrementProduced func(),
) {
	defer utils.PanicHandler(w.cancel)
	logger := log.WithField("function", "fetchTweetOrFailbackToHeapForSimpleWorker")

	logger.WithField("user", entity.Name()).Infoln("fetching user tweets")
	user := heapHelper.GetUserByTwitterId(entity.Uid())
	heap := heapHelper.GetHeap()

	// Helper function to safely push back to heap
	safePushToHeap := func(reason string) {
		logger.WithField("user", entity.Name()).Warnf("%s, pushing back to heap", reason)
		go func() {
			defer func() {
				if r := recover(); r != nil {
					logger.WithField("user", entity.Name()).Errorf("panic while pushing to heap: %v", r)
				}
			}()
			heap.Push(entity)
		}()
	}

	if w.ctx.Err() != nil {
		safePushToHeap("context cancelled")
		return
	}

	logger.WithField("user", entity.Name()).Infof("latest release time: %s", entity.LatestReleaseTime())
	cli := twitter.SelectClientForMediaRequest(w.ctx, clients)
	if cli == nil {
		safePushToHeap("no client available")
		w.cancel(fmt.Errorf("no client available"))
		return
	}

	logger.WithField("user", entity.Name()).Infoln("getting user medias")
	tweets, err := user.GetMeidas(w.ctx, cli, &utils.TimeRange{Min: entity.LatestReleaseTime()})
	if err == twitter.ErrWouldBlock {
		safePushToHeap("client would block")
		return
	}

	logger.WithField("user", entity.Name()).Infoln("got user medias")
	if v, ok := err.(*twitter.TwitterApiError); ok {
		logger.WithField("user", entity.Name()).Warnf("twitter api error: %s", v.Error())
		switch v.Code {
		case twitter.ErrExceedPostLimit:
			twitter.SetClientError(cli, fmt.Errorf("reached the limit for seeing posts today"))
			safePushToHeap("exceed post limit")
			return
		case twitter.ErrAccountLocked:
			twitter.SetClientError(cli, fmt.Errorf("account is locked"))
			safePushToHeap("account locked")
			return
		}
	}

	if w.ctx.Err() != nil {
		safePushToHeap("context cancelled while getting user medias")
		return
	}

	if err != nil {
		logger.WithField("user", entity.Name()).Warnln("failed to get user medias:", err)
		return
	}

	if len(tweets) == 0 {
		logger.WithField("user", entity.Name()).Infoln("no tweets found, updating user medias count")
		if err := database.UpdateUserEntityMediCount(db, entity.Id(), user.MediaCount); err != nil {
			logger.WithField("user", entity.Name()).Panicln("failed to update user medias count:", err)
		}
		return
	}

	logger.WithFields(log.Fields{
		"user":     entity.Name(),
		"tweetNum": len(tweets),
	}).Infoln("found tweets, preparing to push to tweet channel")

	// Push tweets to output channel using SimpleWorker pattern
	pushTimeout := 120 * time.Second
	for _, tw := range tweets {
		pt := dldto.InEntity{Tweet: tw, Entity: entity}

		timeoutTimer := time.NewTimer(pushTimeout)

		select {
		case output <- &pt:
			timeoutTimer.Stop()
			incrementProduced()
			logger.WithField("user", entity.Name()).Infof("pushed tweet %d to tweet channel", tw.Id)
		case <-w.ctx.Done():
			timeoutTimer.Stop()
			logger.WithField("user", entity.Name()).Warnln("context cancelled while pushing tweets")
			return
		case <-timeoutTimer.C:
			logger.WithField("user", entity.Name()).Warnln("timeout while pushing tweet to channel")
			return
		}
	}

	logger.WithField("user", entity.Name()).Infoln("updating user medias count in database")
	if err := database.UpdateUserEntityTweetStat(db, entity.Id(), tweets[0].CreatedAt, user.MediaCount); err != nil {
		logger.WithField("user", entity.Name()).Panicln("failed to update user tweets stat:", err)
	}
}

////////////////////////////////////////////////////////////////////////////////
