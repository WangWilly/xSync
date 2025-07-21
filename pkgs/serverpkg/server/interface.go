package server

import (
	"context"

	"github.com/WangWilly/xSync/pkgs/commonpkg/model"
	"github.com/jmoiron/sqlx"
)

type UserRepo interface {
	GetById(ctx context.Context, db *sqlx.DB, uid uint64) (*model.User, error)
}

type MediaRepo interface {
	GetByUserId(ctx context.Context, db *sqlx.DB, userId uint64) ([]*model.Media, error)
}

type TweetRepo interface {
	GetWithMedia(ctx context.Context, db *sqlx.DB, userId uint64) ([]map[string]interface{}, error)
}
