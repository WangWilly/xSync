package database

import (
	"database/sql"
	"fmt"
	"path/filepath"

	"github.com/jmoiron/sqlx"
)

type User struct {
	Id           uint64 `db:"id"`
	ScreenName   string `db:"screen_name"`
	Name         string `db:"name"`
	IsProtected  bool   `db:"protected"`
	FriendsCount int    `db:"friends_count"`
}

type UserEntity struct {
	Id                sql.NullInt32 `db:"id"`
	Uid               uint64        `db:"user_id"`
	Name              string        `db:"name"`
	LatestReleaseTime sql.NullTime  `db:"latest_release_time"`
	ParentDir         string        `db:"parent_dir"`
	MediaCount        sql.NullInt32 `db:"media_count"`
}

type UserLink struct {
	Id                sql.NullInt32 `db:"id"`
	Uid               uint64        `db:"user_id"`
	Name              string        `db:"name"`
	ParentLstEntityId int32         `db:"parent_lst_entity_id"`
}

type Lst struct {
	Id      uint64 `db:"id"`
	Name    string `db:"name"`
	OwnerId uint64 `db:"owner_uid"`
}

type ListEntity struct {
	Id        sql.NullInt32 `db:"id"`
	LstId     int64         `db:"lst_id"`
	Name      string        `db:"name"`
	ParentDir string        `db:"parent_dir"`
}

func (le *ListEntity) Path() string {
	if le.ParentDir == "" || le.Name == "" {
		panic("no enough info to get path")
	}
	return filepath.Join(le.ParentDir, le.Name)
}

func (ue *UserEntity) Path() string {
	if ue.ParentDir == "" || ue.Name == "" {
		panic("no enough info to get path")
	}
	return filepath.Join(ue.ParentDir, ue.Name)
}

func (ul *UserLink) Path(db *sqlx.DB) (string, error) {
	le, err := GetListEntityById(db, int(ul.ParentLstEntityId))
	if err != nil {
		return "", err
	}
	if le == nil {
		return "", fmt.Errorf("parent lst was not exists")
	}

	return filepath.Join(le.Path(), ul.Name), nil
}
