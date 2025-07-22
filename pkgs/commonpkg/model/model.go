package model

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"time"

	"github.com/jmoiron/sqlx"
)

type User struct {
	Id           uint64    `db:"id"`
	ScreenName   string    `db:"screen_name"`
	Name         string    `db:"name"`
	IsProtected  bool      `db:"protected"`
	FriendsCount int       `db:"friends_count"`
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
}

type UserPreviousName struct {
	Id         int       `db:"id"`
	Uid        uint64    `db:"uid"`
	ScreenName string    `db:"screen_name"`
	Name       string    `db:"name"`
	CreatedAt  time.Time `db:"created_at"`
}

type UserEntity struct {
	Id                sql.NullInt32 `db:"id"`
	Uid               uint64        `db:"user_id"`
	Name              string        `db:"name"`
	ParentDir         string        `db:"parent_dir"`
	FolderName        string        `db:"folder_name"`
	StorageSaved      bool          `db:"storage_saved"`
	MediaCount        sql.NullInt32 `db:"media_count"`
	LatestReleaseTime sql.NullTime  `db:"latest_release_time"`
	CreatedAt         time.Time     `db:"created_at"`
	UpdatedAt         time.Time     `db:"updated_at"`
}

type UserLink struct {
	Id                   sql.NullInt32 `db:"id"`
	UserTwitterId        uint64        `db:"user_id"`
	Name                 string        `db:"name"`
	ListEntityIdBelongTo int32         `db:"parent_lst_entity_id"`
	StorageSaved         bool          `db:"storage_saved"`
	CreatedAt            time.Time     `db:"created_at"`
	UpdatedAt            time.Time     `db:"updated_at"`
}

type List struct {
	Id        uint64    `db:"id"`
	Name      string    `db:"name"`
	OwnerId   uint64    `db:"owner_uid"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

type ListEntity struct {
	Id           sql.NullInt32 `db:"id"`
	LstId        int64         `db:"lst_id"`
	Name         string        `db:"name"`
	ParentDir    string        `db:"parent_dir"`
	FolderName   string        `db:"folder_name"`
	StorageSaved bool          `db:"storage_saved"`
	CreatedAt    time.Time     `db:"created_at"`
	UpdatedAt    time.Time     `db:"updated_at"`
}

type Tweet struct {
	Id        int64     `db:"id"`
	UserId    uint64    `db:"user_id"`
	TweetId   uint64    `db:"tweet_id"`
	Content   string    `db:"content"`
	TweetTime time.Time `db:"tweet_time"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

type Media struct {
	Id        int64     `db:"id"`
	UserId    uint64    `db:"user_id"`
	TweetId   int64     `db:"tweet_id"`
	Location  string    `db:"location"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

func (le *ListEntity) Path() string {
	if le.ParentDir == "" || le.Name == "" {
		panic("no enough info to get path")
	}
	return filepath.Join(le.ParentDir, le.Name)
}

func (ue *UserEntity) Path() string {
	if ue.ParentDir == "" || ue.FolderName == "" {
		panic("no enough info to get path")
	}
	return filepath.Join(ue.ParentDir, ue.FolderName)
}

func (ul *UserLink) Path(db *sqlx.DB) (string, error) {
	// Note: This creates a circular dependency but is needed for backward compatibility
	// In new code, use the repository pattern instead
	stmt := `SELECT * FROM lst_entities WHERE id=?`
	result := &ListEntity{}
	err := db.Get(result, stmt, ul.ListEntityIdBelongTo)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("parent lst was not exists")
		}
		return "", err
	}

	return filepath.Join(result.Path(), ul.Name), nil
}
