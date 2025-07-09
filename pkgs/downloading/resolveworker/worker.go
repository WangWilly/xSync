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

	// config
	pushTimeout time.Duration
}

func NewWorker(mediaDownloadHelper MediaDownloadHelper) *worker {
	return &worker{
		mediaDownloadHelper: mediaDownloadHelper,
		pushTimeout:         120 * time.Second,
	}
}

////////////////////////////////////////////////////////////////////////////////

func (w *worker) ProduceFromHeapToTweetChan(
	ctx context.Context,
	cancel context.CancelCauseFunc,
	heapHelper HeapHelper,
	db *sqlx.DB,
	client *resty.Client,
	additional []*resty.Client,
	output chan<- dldto.TweetDlMeta,
	incrementProduced func(),
) ([]dldto.TweetDlMeta, error) {
	logger := log.WithField("function", "ProduceFromHeap")

	var unsentTweets []dldto.TweetDlMeta

	clients := make([]*resty.Client, 0)
	clients = append(clients, client)
	clients = append(clients, additional...)

	heap := heapHelper.GetHeap()
	logger.WithField("worker", "producer").Infof("initial heap size: %d", heap.Size())

	// Process users from heap sequentially
	for !heap.Empty() && ctx.Err() == nil {
		entity := heap.Peek()
		heap.Pop()

		logger.WithField("user", entity.Name()).Infoln("processing user from heap")
		currUnsentTweets := w.fetchTweetOrFallbackToHeap(ctx, cancel, entity, heapHelper, db, clients, output, incrementProduced)
		if len(currUnsentTweets) > 0 {
			logger.WithField("user", entity.Name()).Warnf("found %d unsent tweets for user, adding to unsent list", len(currUnsentTweets))
			unsentTweets = append(unsentTweets, currUnsentTweets...)
		}
	}

	if ctx.Err() != nil {
		return unsentTweets, ctx.Err()
	}

	logger.WithField("worker", "producer").Info("all producers finished successfully for SimpleWorker")
	return unsentTweets, nil
}

func (w *worker) fetchTweetOrFallbackToHeap(
	ctx context.Context,
	cancel context.CancelCauseFunc,
	entity *smartpathdto.UserSmartPath,
	heapHelper HeapHelper,
	db *sqlx.DB,
	clients []*resty.Client,
	tweetDlMetaOutput chan<- dldto.TweetDlMeta,
	incrementProduced func(),
) []dldto.TweetDlMeta {
	logger := log.WithField("function", "fetchTweetOrFailbackToHeapForSimpleWorker")
	logger.WithField("user", entity.Name()).Infoln("fetching user tweets")

	defer utils.PanicHandler(cancel)

	user := heapHelper.GetUserByTwitterId(entity.TwitterId())
	heap := heapHelper.GetHeap()
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
	tweets, err := user.GetMeidas(ctx, cli, utils.TimeRange{Begin: entity.LatestReleaseTime()})
	if err == twitter.ErrWouldBlock {
		safePushToHeap("client would block")
		return nil
	}
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

	currIdx := 0
tweetLoop:
	for currIdx = range tweets {
		tweetDlMeta := dldto.InEntity{Tweet: tweets[currIdx], Entity: entity}

		timeoutTimer := time.NewTimer(w.pushTimeout)
		select {
		case tweetDlMetaOutput <- &tweetDlMeta:
			timeoutTimer.Stop()
			incrementProduced()
			logger.WithField("user", entity.Name()).Debugf("pushed tweet %d to tweet channel", tweets[currIdx].Id)
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
	for i := currIdx; i < len(tweets); i++ {
		tweetsToUpdate = append(tweetsToUpdate, dldto.InEntity{Tweet: tweets[i], Entity: entity})
	}

	logger.WithField("user", entity.Name()).Infoln("updating user medias count in database")
	if err := database.UpdateUserEntityTweetStat(db, entity.Id(), tweets[0].CreatedAt, user.MediaCount); err != nil {
		logger.WithField("user", entity.Name()).Panicln("failed to update user tweets stat:", err)
	}

	return tweetsToUpdate
}

////////////////////////////////////////////////////////////////////////////////

// DownloadTweetMediaFromTweetChan handles downloading of tweets from a channel
// Returns a slice of tweets that failed to download or were not processed
func (w *worker) DownloadTweetMediaFromTweetChan(
	ctx context.Context,
	cancel context.CancelCauseFunc,
	client *resty.Client,
	tweetDlMetaIn <-chan dldto.TweetDlMeta,
	incrementConsumed func(),
) []dldto.TweetDlMeta {
	logger := log.WithField("function", "DownloadTweetMediaFromTweetChan")

	var failedTweets []dldto.TweetDlMeta

	defer func() {
		if p := recover(); p != nil {
			logger.Errorf("consumer panic: %v", p)
			cancel(fmt.Errorf("consumer panic: %v", p))

			// TODO: not process failedTweets at here
			// Drain remaining tweets and add them to failed list
			drainedCount := 0
			for pt := range tweetDlMetaIn {
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
		case tweetDlMeta, ok := <-tweetDlMetaIn:
			if !ok {
				logger.Debug("tweet channel closed, consumer exiting")
				return failedTweets
			}

			logger.WithField("tweet", tweetDlMeta.GetTweet().Id).Debug("processing tweet")
			err := w.mediaDownloadHelper.SafeDownload(ctx, client, tweetDlMeta)
			incrementConsumed()

			if err == nil {
				logger.WithField("tweet", tweetDlMeta.GetTweet().Id).Debug("downloaded tweet successfully")
				continue
			}

			logger.WithField("tweet", tweetDlMeta.GetTweet().Id).Errorf("failed to download tweet: %v", err)
			failedTweets = append(failedTweets, tweetDlMeta)

			// Cancel context and exit if critical errors occur
			if errors.Is(err, syscall.ENOSPC) {
				logger.Error("no disk space, cancelling context")
				cancel(err)
			}

		case <-ctx.Done():
			// TODO: not process failedTweets at here
			drainedCount := 0
			for pt := range tweetDlMetaIn {
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
