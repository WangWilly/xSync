package downloading

import (
	"context"

	"github.com/WangWilly/xSync/pkgs/commonpkg/clients/twitterclient"
	"github.com/WangWilly/xSync/pkgs/commonpkg/model"
	"github.com/WangWilly/xSync/pkgs/commonpkg/utils"
	"github.com/WangWilly/xSync/pkgs/downloading/dtos/dldto"
	"github.com/WangWilly/xSync/pkgs/downloading/dtos/smartpathdto"
	"github.com/WangWilly/xSync/pkgs/downloading/resolveworker"
	"github.com/jmoiron/sqlx"
)

type UserRepo interface {
	Upsert(db *sqlx.DB, usr *model.User) error
	CreatePreviousName(db *sqlx.DB, uid uint64, name string, screenName string) error
}

type ListRepo interface {
	Upsert(db *sqlx.DB, lst *model.List) error
}

type HeapHelper interface {
	GetDepth(userSmartPath *smartpathdto.UserSmartPath) int
	GetHeap() *utils.Heap[*smartpathdto.UserSmartPath]
	GetUserByTwitterId(twitterId uint64) *twitterclient.User
	MakeHeap(ctx context.Context, db *sqlx.DB, dir string, autoFollow bool) error
}

type DbWorker interface {
	DownloadTweetMediaFromTweetChanWithDB(ctx context.Context, cancel context.CancelCauseFunc, db *sqlx.DB, tweetDlMetaIn <-chan *dldto.NewEntity, incrementConsumed func()) []*dldto.NewEntity
	ProduceFromHeapToTweetChanWithDB(ctx context.Context, cancel context.CancelCauseFunc, heapHelper resolveworker.HeapHelper, db *sqlx.DB, output chan<- *dldto.NewEntity, incrementProduced func()) ([]*dldto.NewEntity, error)
}
