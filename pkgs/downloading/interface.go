package downloading

import (
	"context"

	"github.com/WangWilly/xSync/pkgs/commonpkg/clients/twitterclient"
	"github.com/WangWilly/xSync/pkgs/commonpkg/utils"
	"github.com/WangWilly/xSync/pkgs/downloading/dtos/dldto"
	"github.com/WangWilly/xSync/pkgs/downloading/dtos/smartpathdto"
	"github.com/jmoiron/sqlx"
)

type HeapHelper interface {
	GetDepth(userSmartPath *smartpathdto.UserSmartPath) int
	GetHeap() *utils.Heap[*smartpathdto.UserSmartPath]
	GetUserByTwitterId(twitterId uint64) *twitterclient.User
	MakeHeap(ctx context.Context, db *sqlx.DB, dir string, autoFollow bool) error
}

type DbWorker interface {
	ProduceFromHeapToTweetChanWithDB(ctx context.Context, cancel context.CancelCauseFunc, output chan<- *dldto.NewEntity, incrementProduced func()) ([]*dldto.NewEntity, error)
	DownloadTweetMediaFromTweetChanWithDB(ctx context.Context, cancel context.CancelCauseFunc, tweetDlMetaIn <-chan *dldto.NewEntity, incrementConsumed func()) []*dldto.NewEntity
}
