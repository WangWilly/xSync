package database

import (
	"github.com/WangWilly/xSync/pkgs/model"
	"github.com/jmoiron/sqlx"
)

// Re-export types for backward compatibility
type User = model.User
type UserPreviousName = model.UserPreviousName
type UserEntity = model.UserEntity
type UserLink = model.UserLink
type Lst = model.Lst
type ListEntity = model.ListEntity
type Tweet = model.Tweet
type Media = model.Media

// Re-export CreateTables for backward compatibility
func CreateTables(db *sqlx.DB) {
	model.CreateTables(db)
}
