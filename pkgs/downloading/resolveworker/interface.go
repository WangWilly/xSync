package resolveworker

import (
	"context"
	"time"

	"github.com/WangWilly/xSync/pkgs/commonpkg/clients/twitterclient"
	"github.com/WangWilly/xSync/pkgs/commonpkg/model"
	"github.com/WangWilly/xSync/pkgs/commonpkg/utils"
	"github.com/WangWilly/xSync/pkgs/downloading/dtos/dldto"
	"github.com/WangWilly/xSync/pkgs/downloading/dtos/smartpathdto"
	"github.com/jmoiron/sqlx"
)

type MediaDownloadHelper interface {
	SafeDownload(ctx context.Context, client *twitterclient.Client, meta *dldto.NewEntity) error
}

type HeapHelper interface {
	GetHeap() *utils.Heap[*smartpathdto.UserSmartPath]
	GetDepth(userSmartPath *smartpathdto.UserSmartPath) int
	GetUserByTwitterId(twitterId uint64) *twitterclient.User
}

type UserRepo interface {
	UpdateEntityMediaCount(ctx context.Context, db *sqlx.DB, eid int, count int) error
	UpdateEntityTweetStat(ctx context.Context, db *sqlx.DB, eid int, baseline time.Time, count int) error
}

type TweetRepo interface {
	Create(ctx context.Context, db *sqlx.DB, tweet *model.Tweet) error
	GetByTweetId(ctx context.Context, db *sqlx.DB, tweetId uint64) (*model.Tweet, error)
}

type MediaRepo interface {
	Create(ctx context.Context, db *sqlx.DB, media *model.Media) error
}
