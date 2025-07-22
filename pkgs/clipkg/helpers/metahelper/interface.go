package metahelper

import (
	"context"

	"github.com/WangWilly/xSync/pkgs/commonpkg/clients/twitterclient"
	"github.com/WangWilly/xSync/pkgs/commonpkg/model"
	"github.com/jmoiron/sqlx"
)

type UserRepo interface {
	Upsert(ctx context.Context, db *sqlx.DB, usr *model.User) error
}

type UserEntityRepo interface {
	Upsert(ctx context.Context, db *sqlx.DB, entity *model.UserEntity) error
	GetByTwitterId(ctx context.Context, db *sqlx.DB, twitterId uint64) (*model.UserEntity, error)
	UpdateStorageSavedByTwitterId(ctx context.Context, db *sqlx.DB, twitterId uint64, saved bool) error
}

type LinkRepo interface {
	Upsert(ctx context.Context, db *sqlx.DB, lnk *model.UserLink) error
}

type ListRepo interface {
	Upsert(ctx context.Context, b *sqlx.DB, lst *model.List) error
}

type ListEntityRepo interface {
	Upsert(ctx context.Context, db *sqlx.DB, entity *model.ListEntity) error
	UpdateStorageSavedByTwitterId(ctx context.Context, db *sqlx.DB, twitterId uint64, saved bool) error
}

type PreviousNameRepo interface {
	Create(ctx context.Context, db *sqlx.DB, uid uint64, name string, screenName string) error
}

type ClientManager interface {
	GetMasterClient() *twitterclient.Client
}
