package metahelper

import (
	"github.com/WangWilly/xSync/pkgs/commonpkg/repos/listrepo"
	"github.com/WangWilly/xSync/pkgs/commonpkg/repos/userrepo"
	"github.com/jmoiron/sqlx"
)

type helper struct {
	db       *sqlx.DB
	userRepo UserRepo
	listRepo ListRepo

	twitterClientManager ClientManager
}

func New(db *sqlx.DB, twitterClientManager ClientManager) *helper {
	return &helper{
		db:                   db,
		userRepo:             userrepo.New(),
		listRepo:             listrepo.New(),
		twitterClientManager: twitterClientManager,
	}
}
