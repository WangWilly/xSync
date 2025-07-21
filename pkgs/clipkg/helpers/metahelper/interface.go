package metahelper

import (
	"context"

	"github.com/WangWilly/xSync/pkgs/commonpkg/clients/twitterclient"
	"github.com/WangWilly/xSync/pkgs/commonpkg/model"
	"github.com/jmoiron/sqlx"
)

type UserRepo interface {
	Upsert(ctx context.Context, db *sqlx.DB, usr *model.User) error

	UpsertEntity(ctx context.Context, db *sqlx.DB, entity *model.UserEntity) error
	GetEntityByTwitterId(ctx context.Context, db *sqlx.DB, twitterId uint64) (*model.UserEntity, error)
	UpdateEntityStorageSavedByTwitterId(ctx context.Context, db *sqlx.DB, twitterId uint64, saved bool) error

	CreatePreviousName(ctx context.Context, db *sqlx.DB, uid uint64, name string, screenName string) error

	UpsertLink(ctx context.Context, db *sqlx.DB, lnk *model.UserLink) error
}

type ListRepo interface {
	Upsert(ctx context.Context, b *sqlx.DB, lst *model.List) error

	UpsertEntity(ctx context.Context, db *sqlx.DB, entity *model.ListEntity) error
	UpdateEntityStorageSavedByTwitterId(ctx context.Context, db *sqlx.DB, twitterId uint64, saved bool) error
}

type ClientManager interface {
	GetMasterClient() *twitterclient.Client
}
