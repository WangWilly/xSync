package metahelper

import (
	"github.com/WangWilly/xSync/pkgs/commonpkg/repos/linkrepo"
	"github.com/WangWilly/xSync/pkgs/commonpkg/repos/listentityrepo"
	"github.com/WangWilly/xSync/pkgs/commonpkg/repos/listrepo"
	"github.com/WangWilly/xSync/pkgs/commonpkg/repos/previousnamerepo"
	"github.com/WangWilly/xSync/pkgs/commonpkg/repos/userentityrepo"
	"github.com/WangWilly/xSync/pkgs/commonpkg/repos/userrepo"
	"github.com/jmoiron/sqlx"
)

type helper struct {
	db               *sqlx.DB
	userRepo         UserRepo
	userEntityRepo   UserEntityRepo
	linkRepo         LinkRepo
	listRepo         ListRepo
	listEntityRepo   ListEntityRepo
	previousNameRepo PreviousNameRepo

	twitterClientManager ClientManager
}

func New(db *sqlx.DB, twitterClientManager ClientManager) *helper {
	return &helper{
		db:                   db,
		userRepo:             userrepo.New(),
		userEntityRepo:       userentityrepo.New(),
		linkRepo:             linkrepo.New(),
		listRepo:             listrepo.New(),
		listEntityRepo:       listentityrepo.New(),
		previousNameRepo:     previousnamerepo.New(),
		twitterClientManager: twitterClientManager,
	}
}
