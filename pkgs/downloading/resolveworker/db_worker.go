package resolveworker

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/WangWilly/xSync/pkgs/clients/twitterclient"
	"github.com/WangWilly/xSync/pkgs/database"
	"github.com/WangWilly/xSync/pkgs/downloading/dtos/dldto"
	"github.com/WangWilly/xSync/pkgs/downloading/dtos/smartpathdto"
	"github.com/WangWilly/xSync/pkgs/twitter"
	"github.com/WangWilly/xSync/pkgs/utils"
	"github.com/jmoiron/sqlx"
	log "github.com/sirupsen/logrus"
)

// dbWorker extends the regular worker with database integration for tweets and media
type dbWorker struct {
	pushTimeout          time.Duration
	twitterClientManager *twitterclient.Manager
}

func NewDBWorker(twitterClientManager *twitterclient.Manager) *dbWorker {
	return &dbWorker{
		pushTimeout:          120 * time.Second,
		twitterClientManager: twitterClientManager,
	}
}

////////////////////////////////////////////////////////////////////////////////

// ProduceFromHeapToTweetChanWithDB produces tweets from heap and saves them to database
func (w *dbWorker) ProduceFromHeapToTweetChanWithDB(
	ctx context.Context,
	cancel context.CancelCauseFunc,
	heapHelper HeapHelper,
	db *sqlx.DB,
	output chan<- *dldto.NewEntity,
	incrementProduced func(),
) ([]*dldto.NewEntity, error) {
	logger := log.WithField("function", "ProduceFromHeapWithDB")

	var unsentTweets []*dldto.NewEntity

	heap := heapHelper.GetHeap()
	logger.WithField("worker", "producer").Infof("initial heap size: %d", heap.Size())

	// Process users from heap sequentially
	for !heap.Empty() && ctx.Err() == nil {
		entity := heap.Peek()
		heap.Pop()

		logger.WithField("user", entity.Name()).Infoln("processing user from heap with database integration")
		currUnsentTweets := w.fetchTweetOrFallbackToHeapWithDB(ctx, cancel, entity, heapHelper, db, output, incrementProduced)
		if len(currUnsentTweets) > 0 {
			logger.WithField("user", entity.Name()).Warnf("found %d unsent tweets for user, adding to unsent list", len(currUnsentTweets))
			unsentTweets = append(unsentTweets, currUnsentTweets...)
		}
	}

	if ctx.Err() != nil {
		return unsentTweets, ctx.Err()
	}

	logger.WithField("worker", "producer").Info("all producers finished successfully for SimpleWorker with DB")
	return unsentTweets, nil
}

func (w *dbWorker) fetchTweetOrFallbackToHeapWithDB(
	ctx context.Context,
	cancel context.CancelCauseFunc,
	entity *smartpathdto.UserSmartPath,
	heapHelper HeapHelper,
	db *sqlx.DB,
	tweetDlMetaOutput chan<- *dldto.NewEntity,
	incrementProduced func(),
) []*dldto.NewEntity {
	logger := log.WithField("function", "fetchTweetOrFailbackToHeapWithDB")
	logger.WithField("user", entity.Name()).Infoln("fetching user tweets with database integration")

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
	client := w.twitterClientManager.GetMasterClient()
	if client == nil {
		safePushToHeap("no client available")
		cancel(fmt.Errorf("no client available"))
		return nil
	}
	tweets, err := client.GetMedias(ctx, user, utils.TimeRange{Begin: entity.LatestReleaseTime()})
	if err == twitter.ErrWouldBlock {
		safePushToHeap("client would block")
		return nil
	}
	if v, ok := err.(*twitter.TwitterApiError); ok {
		logger.WithField("user", entity.Name()).Warnf("twitter api error: %s", v.Error())
		switch v.Code {
		case twitter.ErrExceedPostLimit:
			w.twitterClientManager.SetClientError(client, fmt.Errorf("reached the limit for seeing posts today"))
			safePushToHeap("exceed post limit")
			return nil
		case twitter.ErrAccountLocked:
			w.twitterClientManager.SetClientError(client, fmt.Errorf("account is locked"))
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
	}).Infoln("found tweets, saving to database and preparing to push to tweet channel")

	// Save tweets to database before processing
	w.saveTweetsToDatabase(db, tweets, entity.TwitterId(), logger)

	currIdx := 0
tweetLoop:
	for currIdx = range tweets {
		tweetDlMeta := dldto.NewEntity{Tweet: tweets[currIdx], Entity: entity}

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

	var tweetsToUpdate []*dldto.NewEntity
	for i := currIdx; i < len(tweets); i++ {
		tweetsToUpdate = append(tweetsToUpdate, &dldto.NewEntity{Tweet: tweets[i], Entity: entity})
	}

	logger.WithField("user", entity.Name()).Infoln("updating user medias count in database")
	if err := database.UpdateUserEntityTweetStat(db, entity.Id(), tweets[0].CreatedAt, user.MediaCount); err != nil {
		logger.WithField("user", entity.Name()).Panicln("failed to update user tweets stat:", err)
	}

	return tweetsToUpdate
}

// saveTweetsToDatabase saves tweets to the database
func (w *dbWorker) saveTweetsToDatabase(db *sqlx.DB, tweets []*twitterclient.Tweet, userId uint64, logger *log.Entry) {
	for _, tweet := range tweets {
		dbTweet := &database.Tweet{
			UserId:    userId,
			TweetId:   tweet.Id,
			Content:   tweet.Text,
			TweetTime: tweet.CreatedAt,
		}

		if err := database.CreateTweet(db, dbTweet); err != nil {
			logger.
				WithFields(log.Fields{
					"tweet_id": tweet.Id,
					"error":    err,
				}).
				Error("failed to save tweet to database")
			continue
		}

		logger.
			WithFields(log.Fields{
				"tweet_id": tweet.Id,
				"db_id":    dbTweet.Id,
			}).
			Debug("saved tweet to database")
	}
}

////////////////////////////////////////////////////////////////////////////////

// DownloadTweetMediaFromTweetChanWithDB handles downloading of tweets with database integration
func (w *dbWorker) DownloadTweetMediaFromTweetChanWithDB(
	ctx context.Context,
	cancel context.CancelCauseFunc,
	db *sqlx.DB,
	tweetDlMetaIn <-chan *dldto.NewEntity,
	incrementConsumed func(),
) []*dldto.NewEntity {
	logger := log.WithField("function", "DownloadTweetMediaFromTweetChanWithDB")

	var failedTweets []*dldto.NewEntity

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

			logger.WithField("tweet", tweetDlMeta.GetTweet().Id).Debug("processing tweet with DB integration")
			err := w.downloadTweetMediaWithDB(ctx, db, tweetDlMeta, logger)
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

// downloadTweetMediaWithDB downloads media and saves info to database
func (w *dbWorker) downloadTweetMediaWithDB(
	ctx context.Context,
	db *sqlx.DB,
	tweetDlMeta *dldto.NewEntity,
	logger *log.Entry,
) error {
	tweet := tweetDlMeta.GetTweet()

	dbTweet, err := database.GetTweetByTweetId(db, tweet.Id)
	if err != nil {
		logger.WithFields(log.Fields{
			"tweet_id": tweet.Id,
			"user_id":  tweet.Creator.TwitterId,
			"error":    err,
		}).Error("failed to get tweet from database by Twitter ID")
		return err
	}
	if dbTweet == nil {
		logger.WithFields(log.Fields{
			"tweet_id":   tweet.Id,
			"twitter_id": tweet.Id,
			"user_id":    tweet.Creator.TwitterId,
		}).Error("tweet not found in database")
		return fmt.Errorf("tweet with Twitter ID %d not found in database", tweet.Id)
	}

	dbTweetId := dbTweet.Id
	var urls []string
	var mediaRecords []*database.Media
	for i, url := range tweet.Urls {
		// Extract filename from URL or use a generated name
		fileName := filepath.Base(url)
		ext, err := utils.GetExtFromUrl(url)
		if err != nil {
			logger.WithFields(log.Fields{
				"tweet_id": tweet.Id,
				"url":      url,
				"error":    err,
			}).Error("failed to get file extension from URL")
		}
		if ext != "" {
			fileName = fileName + ext
		}
		if fileName == "." || fileName == "/" {
			fileName = fmt.Sprintf("media_%d_%d_%d", tweet.Id, time.Now().Unix(), i)
		}

		// Construct the full path where the media should be saved
		mediaPath := filepath.Join(tweetDlMeta.GetPath(), fileName)
		dbMedia := &database.Media{
			UserId:   tweet.Creator.TwitterId,
			TweetId:  dbTweetId,
			Location: mediaPath,
		}

		if err := database.CreateMedia(db, dbMedia); err != nil {
			logger.WithFields(log.Fields{
				"tweet_id":    tweet.Id,
				"twitter_id":  tweet.Id,
				"db_tweet_id": dbTweetId,
				"media_path":  mediaPath,
				"error":       err,
			}).Error("failed to save media to database")
			continue
		}

		urls = append(urls, url)
		mediaRecords = append(mediaRecords, dbMedia)
		logger.
			WithFields(log.Fields{
				"tweet_id":    tweet.Id,
				"twitter_id":  tweet.Id,
				"db_tweet_id": dbTweetId,
				"media_id":    dbMedia.Id,
				"path":        mediaPath,
			}).
			Debug("created media record in database")
	}

	for i, url := range urls {
		mediaRecord := mediaRecords[i]
		targetPath := mediaRecord.Location

		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			logger.
				WithFields(log.Fields{
					"path":  targetPath,
					"error": err,
				}).
				Error("failed to create directory for media")
			continue
		}

		err := w.twitterClientManager.
			GetMasterClient().
			DownloadToStorageByUrl(ctx, url, targetPath, "4096x4096")
		if err != nil {
			logger.
				WithFields(log.Fields{
					"media_id": mediaRecord.Id,
					"url":      url,
					"target":   targetPath,
					"error":    err,
				}).
				Error("failed to download media file")
		}
	}

	return nil
}
