package downloading

import (
	"context"
	"runtime"

	"github.com/WangWilly/xSync/pkgs/clients/twitterclient"
	"github.com/WangWilly/xSync/pkgs/downloading/dtos/dldto"
	"github.com/WangWilly/xSync/pkgs/downloading/heaphelper"
	"github.com/WangWilly/xSync/pkgs/downloading/mediadownloadhelper"
	"github.com/WangWilly/xSync/pkgs/downloading/resolveworker"
	"github.com/WangWilly/xSync/pkgs/workers"
	"github.com/go-resty/resty/v2"
	"github.com/jmoiron/sqlx"
	log "github.com/sirupsen/logrus"
)

// DownloadHelper encapsulates download functionality with configurable concurrency and client management
type DownloadHelper struct {
	MaxDownloadRoutine int
	Manager            *twitterclient.Manager
}

// NewDownloadHelper creates a new DownloadHelper with default configuration
func NewDownloadHelper() *DownloadHelper {
	return &DownloadHelper{
		MaxDownloadRoutine: min(100, runtime.GOMAXPROCS(0)*10),
		Manager:            twitterclient.NewManager(),
	}
}

// NewDownloadHelperWithConfig creates a new DownloadHelper with custom configuration
func NewDownloadHelperWithConfig(maxRoutines int, manager *twitterclient.Manager) *DownloadHelper {
	if manager == nil {
		manager = twitterclient.NewManager()
	}
	return &DownloadHelper{
		MaxDownloadRoutine: maxRoutines,
		Manager:            manager,
	}
}

// BatchUserDownloadWithDB downloads multiple users with database integration for tweets and media
func (h *DownloadHelper) BatchUserDownloadWithDB(ctx context.Context, client *resty.Client, db *sqlx.DB, users []heaphelper.UserWithinListEntity, dir string, autoFollow bool, additional []*resty.Client) ([]dldto.TweetDlMeta, error) {
	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)

	logger := log.WithField("function", "BatchUserDownloadWithDB")
	logger.Infoln("starting batch user download with database integration")

	// Get additional clients from manager if available
	managedClients := h.Manager.GetAvailableRestyClients()

	// Combine provided additional clients with managed clients
	allAdditionalClients := append(additional, managedClients...)

	heapHelper := heaphelper.NewHelper(users)
	heapHelper.MakeHeap(ctx, db, client, dir, autoFollow)
	mediaDownloadHelper := mediadownloadhelper.NewHelper()
	dbWorker := resolveworker.NewDBWorker(mediaDownloadHelper)
	simpleWorker := workers.NewSimpleWorker[dldto.TweetDlMeta](ctx, cancel, h.MaxDownloadRoutine)

	producer := func(ctx context.Context, cancel context.CancelCauseFunc, output chan<- dldto.TweetDlMeta) ([]dldto.TweetDlMeta, error) {
		return dbWorker.ProduceFromHeapToTweetChanWithDB(ctx, cancel, heapHelper, db, client, allAdditionalClients, output, simpleWorker.IncrementProduced)
	}
	consumer := func(ctx context.Context, cancel context.CancelCauseFunc, input <-chan dldto.TweetDlMeta) []dldto.TweetDlMeta {
		return dbWorker.DownloadTweetMediaFromTweetChanWithDB(ctx, cancel, client, db, input, simpleWorker.IncrementConsumed)
	}

	result := simpleWorker.Process(producer, consumer, h.MaxDownloadRoutine)
	logger.WithFields(log.Fields{
		"produced":        result.Stats.Produced,
		"consumed":        result.Stats.Consumed,
		"failed":          result.Stats.Failed,
		"duration":        result.Stats.Duration,
		"managed_clients": len(managedClients),
		"total_clients":   len(allAdditionalClients),
	}).Info("finished downloading tweets with database integration")

	if result.Error != nil {
		logger.WithError(result.Error).Error("Producer error during tweet download with DB")
		return result.Failed, result.Error
	}
	return result.Failed, context.Cause(ctx)
}

// BatchDownloadTweetWithDB downloads multiple tweets in parallel and returns failed downloads
// 批量下载推文并返回下载失败的推文，可以保证推文被成功下载或被返回
func (h *DownloadHelper) BatchDownloadTweetWithDB(ctx context.Context, client *resty.Client, db *sqlx.DB, tweetDlMetas ...dldto.TweetDlMeta) []dldto.TweetDlMeta {
	if len(tweetDlMetas) == 0 {
		return nil
	}

	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)

	logger := log.WithField("function", "BatchDownloadTweetWithDB")
	logger.WithField("count", len(tweetDlMetas)).Info("starting batch tweet download with DB")

	// Get managed clients for better parallelization
	managedClients := h.Manager.GetAvailableRestyClients()
	allClients := append([]*resty.Client{client}, managedClients...)

	simpleWorker := workers.NewSimpleWorker[dldto.TweetDlMeta](ctx, cancel, min(len(tweetDlMetas), h.MaxDownloadRoutine))
	mediaDownloadHelper := mediadownloadhelper.NewHelper()
	dbWorker := resolveworker.NewDBWorker(mediaDownloadHelper)

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
		return dbWorker.DownloadTweetMediaFromTweetChanWithDB(ctx, cancel, client, db, input, simpleWorker.IncrementConsumed)
	}

	result := simpleWorker.Process(producer, consumer, min(len(tweetDlMetas), h.MaxDownloadRoutine))
	logger.WithFields(log.Fields{
		"produced":        result.Stats.Produced,
		"consumed":        result.Stats.Consumed,
		"failed":          result.Stats.Failed,
		"duration":        result.Stats.Duration,
		"managed_clients": len(managedClients),
		"total_clients":   len(allClients),
	}).Info("completed with DB")

	if result.Error != nil {
		logger.WithError(result.Error).Error("Producer error during tweet download with DB")
	}
	if len(result.Failed) > 0 {
		logger.Warnf("failed to download %d tweets", len(result.Failed))
		return result.Failed
	}
	return nil
}
