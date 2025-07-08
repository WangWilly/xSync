package resolveworker

import (
	"context"
	"errors"
	"fmt"
	"syscall"
	"time"

	"github.com/WangWilly/xSync/pkgs/database"
	"github.com/WangWilly/xSync/pkgs/downloading/dtos/dldto"
	"github.com/WangWilly/xSync/pkgs/downloading/dtos/smartpathdto"
	"github.com/WangWilly/xSync/pkgs/twitter"
	"github.com/WangWilly/xSync/pkgs/utils"
	"github.com/go-resty/resty/v2"
	"github.com/jmoiron/sqlx"
	log "github.com/sirupsen/logrus"
)

type worker struct {
	mediaDownloadHelper MediaDownloadHelper
}

func NewWorker(mediaDownloadHelper MediaDownloadHelper) *worker {
	return &worker{
		mediaDownloadHelper: mediaDownloadHelper,
	}
}

////////////////////////////////////////////////////////////////////////////////

// DownloadTweetMediaFromTweetChan handles downloading of tweets from a channel
// Returns a slice of tweets that failed to download or were not processed
func (w *worker) DownloadTweetMediaFromTweetChan(
	ctx context.Context,
	cancel context.CancelCauseFunc,
	client *resty.Client,
	tweetChan <-chan dldto.TweetDlMeta,
	incrementConsumed func(),
) []dldto.TweetDlMeta {
	logger := log.WithField("function", "DownloadTweetMediaFromTweetChan")
	logger.Debug("consumer started")

	var failedTweets []dldto.TweetDlMeta

	defer func() {
		logger.WithField("failedCount", len(failedTweets)).Debug("consumer finished")
	}()

	defer func() {
		if p := recover(); p != nil {
			logger.Errorf("consumer panic: %v", p)
			cancel(fmt.Errorf("consumer panic: %v", p))

			// Drain remaining tweets and add them to failed list
			drainedCount := 0
			for pt := range tweetChan {
				incrementConsumed()
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
			err := w.mediaDownloadHelper.SafeDownload(ctx, client, pt)
			incrementConsumed()

			if err != nil {
				logger.WithField("tweet", pt.GetTweet().Id).Errorf("failed to download tweet: %v", err)
				failedTweets = append(failedTweets, pt)

				// Cancel context and exit if critical errors occur
				if errors.Is(err, syscall.ENOSPC) {
					logger.Error("no disk space, cancelling context")
					cancel(err)
				}
				return failedTweets
			} else {
				logger.WithField("tweet", pt.GetTweet().Id).Debug("downloaded tweet successfully")
			}

		case <-ctx.Done():
			logger.Debug("context cancelled, draining remaining tweets")
			// Drain remaining tweets and add them to failed list
			drainedCount := 0
			for pt := range tweetChan {
				incrementConsumed()
				drainedCount++
				failedTweets = append(failedTweets, pt)
				logger.WithField("tweet", pt.GetTweet().Id).Debug("added drained tweet to failed list")
			}
			logger.WithField("drainedCount", drainedCount).Debug("finished draining tweets due to context cancellation")
			return failedTweets
		}
	}
}

////////////////////////////////////////////////////////////////////////////////

// ProduceFromHeap produces tweets from heap for SimpleWorker pattern
func (w *worker) ProduceFromHeap(
	ctx context.Context,
	cancel context.CancelCauseFunc,
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
	for !heap.Empty() && ctx.Err() == nil {
		entity := heap.Peek()
		heap.Pop()

		log.WithField("user", entity.Name()).Debug("processing user from heap")

		currUnsentTweets := w.fetchTweetOrFailbackToHeapForSimpleWorker(ctx, cancel, entity, heapHelper, db, clients, output, incrementProduced)
		if len(currUnsentTweets) > 0 {
			log.WithField("user", entity.Name()).Warnf("found %d unsent tweets for user, adding to unsent list", len(currUnsentTweets))
			unsentTweets = append(unsentTweets, currUnsentTweets...)
		}
	}

	if ctx.Err() != nil {
		return unsentTweets, ctx.Err()
	}

	log.WithField("worker", "producer").Info("all producers finished successfully for SimpleWorker")
	return unsentTweets, nil
}

// fetchTweetOrFailbackToHeapForSimpleWorker is adapted for SimpleWorker pattern
func (w *worker) fetchTweetOrFailbackToHeapForSimpleWorker(
	ctx context.Context,
	cancel context.CancelCauseFunc,
	entity *smartpathdto.UserSmartPath,
	heapHelper HeapHelper,
	db *sqlx.DB,
	clients []*resty.Client,
	output chan<- dldto.TweetDlMeta,
	incrementProduced func(),
) []dldto.TweetDlMeta {
	defer utils.PanicHandler(cancel)
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

	if ctx.Err() != nil {
		safePushToHeap("context cancelled")
		return nil
	}

	logger.WithField("user", entity.Name()).Infof("latest release time: %s", entity.LatestReleaseTime())
	cli := twitter.SelectClientForMediaRequest(ctx, clients)
	if cli == nil {
		safePushToHeap("no client available")
		cancel(fmt.Errorf("no client available"))
		return nil
	}

	logger.WithField("user", entity.Name()).Infoln("getting user medias")
	tweets, err := user.GetMeidas(ctx, cli, &utils.TimeRange{Min: entity.LatestReleaseTime()})
	if err == twitter.ErrWouldBlock {
		safePushToHeap("client would block")
		return nil
	}

	logger.WithField("user", entity.Name()).Infoln("got user medias")
	if v, ok := err.(*twitter.TwitterApiError); ok {
		logger.WithField("user", entity.Name()).Warnf("twitter api error: %s", v.Error())
		switch v.Code {
		case twitter.ErrExceedPostLimit:
			twitter.SetClientError(cli, fmt.Errorf("reached the limit for seeing posts today"))
			safePushToHeap("exceed post limit")
			return nil
		case twitter.ErrAccountLocked:
			twitter.SetClientError(cli, fmt.Errorf("account is locked"))
			safePushToHeap("account locked")
			return nil
		}
	}

	if ctx.Err() != nil {
		safePushToHeap("context cancelled while getting user medias")
		return nil
	}

	if err != nil {
		logger.WithField("user", entity.Name()).Warnln("failed to get user medias:", err)
		return nil
	}

	if len(tweets) == 0 {
		logger.WithField("user", entity.Name()).Infoln("no tweets found, updating user medias count")
		if err := database.UpdateUserEntityMediCount(db, entity.Id(), user.MediaCount); err != nil {
			logger.WithField("user", entity.Name()).Panicln("failed to update user medias count:", err)
		}
		return nil
	}

	logger.WithFields(log.Fields{
		"user":     entity.Name(),
		"tweetNum": len(tweets),
	}).Infoln("found tweets, preparing to push to tweet channel")

	// Push tweets to output channel using SimpleWorker pattern
	pushTimeout := 120 * time.Second
	idx := 0
tweetLoop:
	for idx = range tweets {
		pt := dldto.InEntity{Tweet: tweets[idx], Entity: entity}

		timeoutTimer := time.NewTimer(pushTimeout)

		select {
		case output <- &pt:
			timeoutTimer.Stop()
			incrementProduced()
			logger.WithField("user", entity.Name()).Infof("pushed tweet %d to tweet channel", tweets[idx].Id)
		case <-ctx.Done():
			timeoutTimer.Stop()
			logger.WithField("user", entity.Name()).Warnln("context cancelled while pushing tweet to channel")
			break tweetLoop
		case <-timeoutTimer.C:
			logger.WithField("user", entity.Name()).Warnln("timeout while pushing tweet to channel")
			break tweetLoop
		}
	}

	var tweetsToUpdate []dldto.TweetDlMeta
	for ; idx < len(tweets); idx++ {
		tweetsToUpdate = append(tweetsToUpdate, dldto.InEntity{Tweet: tweets[idx], Entity: entity})
	}

	logger.WithField("user", entity.Name()).Infoln("updating user medias count in database")
	if err := database.UpdateUserEntityTweetStat(db, entity.Id(), tweets[0].CreatedAt, user.MediaCount); err != nil {
		logger.WithField("user", entity.Name()).Panicln("failed to update user tweets stat:", err)
	}

	return tweetsToUpdate
}

////////////////////////////////////////////////////////////////////////////////
