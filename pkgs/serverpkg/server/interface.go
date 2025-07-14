package server

import (
	"github.com/WangWilly/xSync/pkgs/commonpkg/model"
	"github.com/jmoiron/sqlx"
)

type UserRepo interface {
	GetById(db *sqlx.DB, uid uint64) (*model.User, error)
}

type MediaRepo interface {
	GetByUserId(db *sqlx.DB, userId uint64) ([]*model.Media, error)
}

type TweetRepo interface {
	GetWithMedia(db *sqlx.DB, userId uint64) ([]map[string]interface{}, error)
}
