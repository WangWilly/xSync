package heaphelper

import (
	"time"

	"github.com/WangWilly/xSync/pkgs/commonpkg/model"
	"github.com/jmoiron/sqlx"
)

type UserRepo interface {
	Create(db *sqlx.DB, usr *model.User) error
	CreateEntity(db *sqlx.DB, entity *model.UserEntity) error
	CreateLink(db *sqlx.DB, lnk *model.UserLink) error
	Delete(db *sqlx.DB, uid uint64) error
	DeleteEntity(db *sqlx.DB, id uint32) error
	DeleteLink(db *sqlx.DB, id int32) error
	GetById(db *sqlx.DB, uid uint64) (*model.User, error)
	GetEntity(db *sqlx.DB, uid uint64, parentDir string) (*model.UserEntity, error)
	GetEntityById(db *sqlx.DB, id int) (*model.UserEntity, error)
	GetLink(db *sqlx.DB, uid uint64, parentLstEntityId int32) (*model.UserLink, error)
	GetLinks(db *sqlx.DB, uid uint64) ([]*model.UserLink, error)
	RecordPreviousName(db *sqlx.DB, uid uint64, name string, screenName string) error
	SetEntityLatestReleaseTime(db *sqlx.DB, id int, t time.Time) error
	Update(db *sqlx.DB, usr *model.User) error
	UpdateEntity(db *sqlx.DB, entity *model.UserEntity) error
	UpdateEntityMediaCount(db *sqlx.DB, eid int, count int) error
	UpdateEntityTweetStat(db *sqlx.DB, eid int, baseline time.Time, count int) error
	UpdateLink(db *sqlx.DB, id int32, name string) error
}

type ListRepo interface {
	Create(db *sqlx.DB, lst *model.List) error
	CreateEntity(db *sqlx.DB, entity *model.ListEntity) error
	Delete(db *sqlx.DB, lid uint64) error
	DeleteEntity(db *sqlx.DB, id int) error
	GetById(db *sqlx.DB, lid uint64) (*model.List, error)
	GetEntity(db *sqlx.DB, lid int64, parentDir string) (*model.ListEntity, error)
	GetEntityById(db *sqlx.DB, id int) (*model.ListEntity, error)
	Update(db *sqlx.DB, lst *model.List) error
	UpdateEntity(db *sqlx.DB, entity *model.ListEntity) error
}
