package heaphelper

import (
	"context"

	"github.com/WangWilly/xSync/pkgs/commonpkg/model"
	"github.com/jmoiron/sqlx"
)

type UserRepo interface {
	CreateLink(ctx context.Context, db *sqlx.DB, entity *model.UserLink) error
	GetLinks(ctx context.Context, db *sqlx.DB, userId uint64) ([]*model.UserLink, error)
}

type ListRepo interface {
	Create(ctx context.Context, db *sqlx.DB, lst *model.List) error
	GetById(ctx context.Context, db *sqlx.DB, id uint64) (*model.List, error)
	Update(ctx context.Context, db *sqlx.DB, lst *model.List) error
}
