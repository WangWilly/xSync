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

var (
	MaxDownloadRoutine int
)

func init() {
	MaxDownloadRoutine = min(100, runtime.GOMAXPROCS(0)*10)
}

////////////////////////////////////////////////////////////////////////////////

func BatchUserDownload(ctx context.Context, client *resty.Client, db *sqlx.DB, users []heaphelper.UserWithinListEntity, dir string, autoFollow bool, additional []*resty.Client) ([]dldto.TweetDlMeta, error) {
	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)

	logger := log.WithField("function", "BatchUserDownload")
	logger.Infoln("starting batch user download")

	heapHelper := heaphelper.NewHelper(users)
	heapHelper.MakeHeap(ctx, db, client, dir, autoFollow)
	mediaDownloadHelper := mediadownloadhelper.NewHelper()
	simpleWorker := workers.NewSimpleWorker[dldto.TweetDlMeta](ctx, cancel, MaxDownloadRoutine)
	worker := resolveworker.NewWorker(mediaDownloadHelper)

	producer := func(ctx context.Context, cancel context.CancelCauseFunc, output chan<- dldto.TweetDlMeta) ([]dldto.TweetDlMeta, error) {
		return worker.ProduceFromHeapToTweetChan(ctx, cancel, heapHelper, db, client, additional, output, simpleWorker.IncrementProduced)
	}
	consumer := func(ctx context.Context, cancel context.CancelCauseFunc, input <-chan dldto.TweetDlMeta) []dldto.TweetDlMeta {
		return worker.DownloadTweetMediaFromTweetChan(ctx, cancel, client, input, simpleWorker.IncrementConsumed)
	}

	result := simpleWorker.Process(producer, consumer, MaxDownloadRoutine)
	logger.WithFields(log.Fields{
		"produced": result.Stats.Produced,
		"consumed": result.Stats.Consumed,
		"failed":   result.Stats.Failed,
		"duration": result.Stats.Duration,
	}).Info("finished downloading tweets")

	if result.Error != nil {
		logger.WithError(result.Error).Error("Producer error during tweet download")
		return result.Failed, result.Error
	}
	return result.Failed, context.Cause(ctx)
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

	logger := log.WithField("function", "BatchDownloadTweet")
	logger.WithField("count", len(tweetDlMetas)).Info("starting batch tweet download")

	simpleWorker := workers.NewSimpleWorker[dldto.TweetDlMeta](ctx, cancel, min(len(tweetDlMetas), MaxDownloadRoutine))
	mediaDownloadHelper := mediadownloadhelper.NewHelper()
	worker := resolveworker.NewWorker(mediaDownloadHelper)

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
	consumer := func(ctx context.Context, cancel context.CancelCauseFunc, input <-chan dldto.TweetDlMeta) []dldto.TweetDlMeta {
		return worker.DownloadTweetMediaFromTweetChan(ctx, cancel, client, input, simpleWorker.IncrementConsumed)
	}

	result := simpleWorker.Process(producer, consumer, min(len(tweetDlMetas), MaxDownloadRoutine))
	logger.WithFields(log.Fields{
		"produced": result.Stats.Produced,
		"consumed": result.Stats.Consumed,
		"failed":   result.Stats.Failed,
		"duration": result.Stats.Duration,
	}).Info("completed")

	if result.Error != nil {
		logger.WithError(result.Error).Error("Producer error during tweet download")
	}
	if len(result.Failed) > 0 {
		logger.Warnf("failed to download %d tweets", len(result.Failed))
		return result.Failed
	}
	return nil
}
