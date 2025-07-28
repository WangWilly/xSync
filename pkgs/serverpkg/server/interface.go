package server

import (
	"context"

	"github.com/WangWilly/xSync/pkgs/commonpkg/model"
	"github.com/jmoiron/sqlx"
)

type UserRepo interface {
	GetById(ctx context.Context, db *sqlx.DB, uid uint64) (*model.User, error)
	ListAll(ctx context.Context, db *sqlx.DB) ([]*model.User, error)
}

type UserEntityRepo interface {
	GetByTwitterId(ctx context.Context, db *sqlx.DB, twitterId uint64) (*model.UserEntity, error)
}

type MediaRepo interface {
	ListByUserId(ctx context.Context, db *sqlx.DB, userId uint64) ([]*model.Media, error)
	CountAll(ctx context.Context, db *sqlx.DB) (int64, error)
	CountByUserId(ctx context.Context, db *sqlx.DB, userId uint64) (int64, error)
}

type TweetRepo interface {
	GetWithMedia(ctx context.Context, db *sqlx.DB, userId uint64) ([]map[string]interface{}, error)
	ListByUserId(context.Context, *sqlx.DB, uint64) ([]*model.Tweet, error)
	CountAll(ctx context.Context, db *sqlx.DB) (int64, error)
	CountByUserId(ctx context.Context, db *sqlx.DB, userId uint64) (int64, error)
}
