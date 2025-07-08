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
	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)

	heapHelper := heaphelper.NewHelper(users)
	heapHelper.MakeHeap(ctx, db, client, dir, autoFollow)
	log.Infof("heap size: %d\n", heapHelper.GetHeap().Size())

	mediaDownloadHelper := mediadownloadhelper.NewHelper()
	log.Infoln("start downloading tweets")

	// Create SimpleWorker for tweet media downloading
	simpleWorker := workers.NewSimpleWorker[dldto.TweetDlMeta](ctx, cancel, MaxDownloadRoutine)

	// Create the original worker for heap processing and tweet downloading logic
	worker := resolveworker.NewWorker(mediaDownloadHelper)

	// Define producer function that processes the heap
	producer := func(ctx context.Context, cancel context.CancelCauseFunc, output chan<- dldto.TweetDlMeta) ([]dldto.TweetDlMeta, error) {
		return worker.ProduceFromHeap(ctx, cancel, heapHelper, db, client, additional, output, simpleWorker.IncrementProduced)
	}

	// Define consumer function that downloads tweet media
	consumer := func(ctx context.Context, cancel context.CancelCauseFunc, input <-chan dldto.TweetDlMeta) []dldto.TweetDlMeta {
		return worker.DownloadTweetMediaFromTweetChan(ctx, cancel, client, input, simpleWorker.IncrementConsumed)
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

	simpleWorker := workers.NewSimpleWorker[dldto.TweetDlMeta](ctx, cancel, min(len(tweetDlMetas), MaxDownloadRoutine))

	mediaDownloadHelper := mediadownloadhelper.NewHelper()
	worker := resolveworker.NewWorker(mediaDownloadHelper)

	// Define producer function that sends the tweet list
	producer := func(ctx context.Context, cancel context.CancelCauseFunc, output chan<- dldto.TweetDlMeta) ([]dldto.TweetDlMeta, error) {
		idx := 0
	tweetDlMetasLoop:
		for idx := range tweetDlMetas {
			select {
			case <-ctx.Done():
				break tweetDlMetasLoop
			case output <- tweetDlMetas[idx]:
				simpleWorker.IncrementProduced()
			}
		}
		var unsent []dldto.TweetDlMeta
		for ; idx < len(tweetDlMetas); idx++ {
			unsent = append(unsent, tweetDlMetas[idx])
		}
		return unsent, nil
	}

	// Define consumer function that downloads tweet media
	consumer := func(ctx context.Context, cancel context.CancelCauseFunc, input <-chan dldto.TweetDlMeta) []dldto.TweetDlMeta {
		return worker.DownloadTweetMediaFromTweetChan(ctx, cancel, client, input, simpleWorker.IncrementConsumed)
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
