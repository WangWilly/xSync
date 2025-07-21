package metahelper

import (
	"github.com/WangWilly/xSync/pkgs/commonpkg/clients/twitterclient"
	"github.com/WangWilly/xSync/pkgs/commonpkg/model"
	"github.com/jmoiron/sqlx"
)

type UserRepo interface {
	Upsert(db *sqlx.DB, usr *model.User) error

	UpsertEntity(db *sqlx.DB, entity *model.UserEntity) error
	GetEntityByTwitterId(db *sqlx.DB, twitterId uint64) (*model.UserEntity, error)
	UpdateEntityStorageSavedByTwitterId(db *sqlx.DB, twitterId uint64, saved bool) error

	CreatePreviousName(db *sqlx.DB, uid uint64, name string, screenName string) error

	UpsertLink(db *sqlx.DB, lnk *model.UserLink) error
}

type ListRepo interface {
	Upsert(db *sqlx.DB, lst *model.List) error

	UpsertEntity(db *sqlx.DB, entity *model.ListEntity) error
	UpdateEntityStorageSavedByTwitterId(db *sqlx.DB, twitterId uint64, saved bool) error
}

type ClientManager interface {
	GetMasterClient() *twitterclient.Client
}
