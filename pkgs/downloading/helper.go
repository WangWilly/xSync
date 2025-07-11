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
	"github.com/jmoiron/sqlx"
	log "github.com/sirupsen/logrus"
)

type Config struct {
	MaxDownloadRoutine int

	DownloadDir string
	AutoFollow  bool
}

type helper struct {
	cfg Config

	twitterClientManager *twitterclient.Manager
	heapHelper           HeapHelper
}

func NewDownloadHelperWithConfig(cfg Config, manager *twitterclient.Manager, heaphelper HeapHelper) *helper {
	defaultMaxDownloadRoutine := min(100, runtime.GOMAXPROCS(0)*10)
	if cfg.MaxDownloadRoutine <= 0 {
		cfg.MaxDownloadRoutine = defaultMaxDownloadRoutine
	}

	return &helper{
		cfg:                  cfg,
		twitterClientManager: manager,
		heapHelper:           heaphelper,
	}
}

func (h *helper) BatchUserDownloadWithDB(ctx context.Context, db *sqlx.DB, users []heaphelper.UserWithinListEntity) ([]*dldto.NewEntity, error) {
	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)

	logger := log.WithField("function", "BatchUserDownloadWithDB")
	logger.Infoln("starting batch user download with database integration")

	h.heapHelper.MakeHeap(ctx, db, h.cfg.DownloadDir, h.cfg.AutoFollow)

	mediaDownloadHelper := mediadownloadhelper.NewHelper()
	dbWorker := resolveworker.NewDBWorker(mediaDownloadHelper, h.twitterClientManager)
	simpleWorker := workers.NewSimpleWorker[*dldto.NewEntity](ctx, cancel, h.cfg.MaxDownloadRoutine)

	producer := func(ctx context.Context, cancel context.CancelCauseFunc, output chan<- *dldto.NewEntity) ([]*dldto.NewEntity, error) {
		return dbWorker.ProduceFromHeapToTweetChanWithDB(ctx, cancel, h.heapHelper, db, output, simpleWorker.IncrementProduced)
	}
	consumer := func(ctx context.Context, cancel context.CancelCauseFunc, input <-chan *dldto.NewEntity) []*dldto.NewEntity {
		return dbWorker.DownloadTweetMediaFromTweetChanWithDB(ctx, cancel, db, input, simpleWorker.IncrementConsumed)
	}

	result := simpleWorker.Process(producer, consumer, h.cfg.MaxDownloadRoutine)
	logger.WithFields(log.Fields{
		"produced": result.Stats.Produced,
		"consumed": result.Stats.Consumed,
		"failed":   result.Stats.Failed,
		"duration": result.Stats.Duration,
	}).Info("finished downloading tweets with database integration")

	if result.Error != nil {
		logger.WithError(result.Error).Error("Producer error during tweet download with DB")
		return result.Failed, result.Error
	}
	return result.Failed, context.Cause(ctx)
}

// BatchDownloadTweetWithDB downloads multiple tweets in parallel and returns failed downloads
// 批量下载推文并返回下载失败的推文，可以保证推文被成功下载或被返回
func (h *helper) BatchDownloadTweetWithDB(ctx context.Context, db *sqlx.DB, tweetDlMetas ...*dldto.NewEntity) []*dldto.NewEntity {
	if len(tweetDlMetas) == 0 {
		return nil
	}

	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)

	logger := log.WithField("function", "BatchDownloadTweetWithDB")
	logger.WithField("count", len(tweetDlMetas)).Info("starting batch tweet download with DB")

	mediaDownloadHelper := mediadownloadhelper.NewHelper()
	dbWorker := resolveworker.NewDBWorker(mediaDownloadHelper, h.twitterClientManager)
	simpleWorker := workers.NewSimpleWorker[*dldto.NewEntity](ctx, cancel, min(len(tweetDlMetas), h.cfg.MaxDownloadRoutine))

	producer := func(ctx context.Context, cancel context.CancelCauseFunc, output chan<- *dldto.NewEntity) ([]*dldto.NewEntity, error) {
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
		var unsent []*dldto.NewEntity
		for ; idx < len(tweetDlMetas); idx++ {
			unsent = append(unsent, tweetDlMetas[idx])
		}
		return unsent, nil
	}
	consumer := func(ctx context.Context, cancel context.CancelCauseFunc, input <-chan *dldto.NewEntity) []*dldto.NewEntity {
		return dbWorker.DownloadTweetMediaFromTweetChanWithDB(ctx, cancel, db, input, simpleWorker.IncrementConsumed)
	}

	result := simpleWorker.Process(producer, consumer, min(len(tweetDlMetas), h.cfg.MaxDownloadRoutine))
	logger.WithFields(log.Fields{
		"produced": result.Stats.Produced,
		"consumed": result.Stats.Consumed,
		"failed":   result.Stats.Failed,
		"duration": result.Stats.Duration,
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
