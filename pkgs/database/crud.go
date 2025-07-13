package database

import (
	"time"

	"github.com/WangWilly/xSync/pkgs/model"
	"github.com/WangWilly/xSync/pkgs/repos/listrepo"
	"github.com/WangWilly/xSync/pkgs/repos/mediarepo"
	"github.com/WangWilly/xSync/pkgs/repos/tweetrepo"
	"github.com/WangWilly/xSync/pkgs/repos/userrepo"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

// Global repository instances for backward compatibility
var (
	userRepo  = userrepo.New()
	listRepo  = listrepo.New()
	tweetRepo = tweetrepo.New()
	mediaRepo = mediarepo.New()
)

// Deprecated: Use userRepo directly for CRUD operations
//
// User CRUD operations - backward compatibility wrappers
func CreateUser(db *sqlx.DB, usr *model.User) error {
	return userRepo.Create(db, usr)
}

// Deprecated: Use userRepo directly for CRUD operations
func DelUser(db *sqlx.DB, uid uint64) error {
	return userRepo.Delete(db, uid)
}

// Deprecated: Use userRepo directly for CRUD operations
func GetUserById(db *sqlx.DB, uid uint64) (*model.User, error) {
	return userRepo.GetById(db, uid)
}

// Deprecated: Use userRepo directly for CRUD operations
func UpdateUser(db *sqlx.DB, usr *model.User) error {
	return userRepo.Update(db, usr)
}

// Deprecated: Use userRepo directly for CRUD operations
//
// UserEntity CRUD operations - backward compatibility wrappers
func CreateUserEntity(db *sqlx.DB, entity *model.UserEntity) error {
	return userRepo.CreateEntity(db, entity)
}

// Deprecated: Use userRepo directly for CRUD operations
func DelUserEntity(db *sqlx.DB, id uint32) error {
	return userRepo.DeleteEntity(db, id)
}

// Deprecated: Use userRepo directly for CRUD operations
func GetUserEntity(db *sqlx.DB, uid uint64, parentDir string) (*model.UserEntity, error) {
	return userRepo.GetEntity(db, uid, parentDir)
}

// Deprecated: Use userRepo directly for CRUD operations
func GetUserEntityById(db *sqlx.DB, id int) (*model.UserEntity, error) {
	return userRepo.GetEntityById(db, id)
}

// Deprecated: Use userRepo directly for CRUD operations
func UpdateUserEntity(db *sqlx.DB, entity *model.UserEntity) error {
	return userRepo.UpdateEntity(db, entity)
}

// Deprecated: Use userRepo directly for CRUD operations
func UpdateUserEntityMediCount(db *sqlx.DB, eid int, count int) error {
	return userRepo.UpdateEntityMediaCount(db, eid, count)
}

// Deprecated: Use userRepo directly for CRUD operations
func UpdateUserEntityTweetStat(db *sqlx.DB, eid int, baseline time.Time, count int) error {
	return userRepo.UpdateEntityTweetStat(db, eid, baseline, count)
}

// Deprecated: Use userRepo directly for CRUD operations
func SetUserEntityLatestReleaseTime(db *sqlx.DB, id int, t time.Time) error {
	return userRepo.SetEntityLatestReleaseTime(db, id, t)
}

// Deprecated: Use userRepo directly for CRUD operations
func RecordUserPreviousName(db *sqlx.DB, uid uint64, name string, screenName string) error {
	return userRepo.RecordPreviousName(db, uid, name, screenName)
}

// Deprecated: Use userRepo directly for CRUD operations
// model.UserLink CRUD operations - backward compatibility wrappers
func CreateUserLink(db *sqlx.DB, lnk *model.UserLink) error {
	return userRepo.CreateLink(db, lnk)
}

// Deprecated: Use userRepo directly for CRUD operations
func DelUserLink(db *sqlx.DB, id int32) error {
	return userRepo.DeleteLink(db, id)
}

// Deprecated: Use userRepo directly for CRUD operations
func GetUserLinks(db *sqlx.DB, uid uint64) ([]*model.UserLink, error) {
	return userRepo.GetLinks(db, uid)
}

// Deprecated: Use userRepo directly for CRUD operations
func GetUserLink(db *sqlx.DB, uid uint64, parentLstEntityId int32) (*model.UserLink, error) {
	return userRepo.GetLink(db, uid, parentLstEntityId)
}

// Deprecated: Use userRepo directly for CRUD operations
func UpdateUserLink(db *sqlx.DB, id int32, name string) error {
	return userRepo.UpdateLink(db, id, name)
}

// Deprecated: Use listRepo directly for CRUD operations
//
// List CRUD operations - backward compatibility wrappers
func CreateLst(db *sqlx.DB, lst *model.List) error {
	return listRepo.Create(db, lst)
}

// Deprecated: Use listRepo directly for CRUD operations
func DelLst(db *sqlx.DB, lid uint64) error {
	return listRepo.Delete(db, lid)
}

// Deprecated: Use listRepo directly for CRUD operations
func GetLst(db *sqlx.DB, lid uint64) (*model.List, error) {
	return listRepo.GetById(db, lid)
}

// Deprecated: Use listRepo directly for CRUD operations
func UpdateLst(db *sqlx.DB, lst *model.List) error {
	return listRepo.Update(db, lst)
}

// Deprecated: Use listRepo directly for CRUD operations
// model.ListEntity CRUD operations - backward compatibility wrappers
func CreateLstEntity(db *sqlx.DB, entity *model.ListEntity) error {
	return listRepo.CreateEntity(db, entity)
}

// Deprecated: Use listRepo directly for CRUD operations
func DelLstEntity(db *sqlx.DB, id int) error {
	return listRepo.DeleteEntity(db, id)
}

// Deprecated: Use listRepo directly for CRUD operations
func GetListEntityById(db *sqlx.DB, id int) (*model.ListEntity, error) {
	return listRepo.GetEntityById(db, id)
}

// Deprecated: Use listRepo directly for CRUD operations
func GetListEntity(db *sqlx.DB, lid int64, parentDir string) (*model.ListEntity, error) {
	return listRepo.GetEntity(db, lid, parentDir)
}

// Deprecated: Use listRepo directly for CRUD operations
func UpdateLstEntity(db *sqlx.DB, entity *model.ListEntity) error {
	return listRepo.UpdateEntity(db, entity)
}

// Deprecated: Use tweetRepo directly for CRUD operations
//
// model.Tweet CRUD operations - backward compatibility wrappers
func CreateTweet(db *sqlx.DB, tweet *model.Tweet) error {
	return tweetRepo.Create(db, tweet)
}

// Deprecated: Use tweetRepo directly for CRUD operations
func GetTweetById(db *sqlx.DB, id int64) (*model.Tweet, error) {
	return tweetRepo.GetById(db, id)
}

// Deprecated: Use tweetRepo directly for CRUD operations
func GetTweetsByUserId(db *sqlx.DB, userId uint64) ([]*model.Tweet, error) {
	return tweetRepo.GetByUserId(db, userId)
}

// Deprecated: Use tweetRepo directly for CRUD operations
func UpdateTweet(db *sqlx.DB, tweet *model.Tweet) error {
	return tweetRepo.Update(db, tweet)
}

// Deprecated: Use tweetRepo directly for CRUD operations
func DeleteTweet(db *sqlx.DB, id int64) error {
	return tweetRepo.Delete(db, id)
}

// Deprecated: Use tweetRepo directly for CRUD operations
func GetTweetByTweetId(db *sqlx.DB, twitterId uint64) (*model.Tweet, error) {
	return tweetRepo.GetByTweetId(db, twitterId)
}

// Deprecated: Use tweetRepo directly for CRUD operations
//
// Helper functions for tweet-media relationships
func GetTweetsWithMedia(db *sqlx.DB, userId uint64) ([]map[string]interface{}, error) {
	return tweetRepo.GetWithMedia(db, userId)
}

// Deprecated: Use mediaRepo directly for CRUD operations
//
// model.Media CRUD operations - backward compatibility wrappers
func CreateMedia(db *sqlx.DB, media *model.Media) error {
	return mediaRepo.Create(db, media)
}

// Deprecated: Use mediaRepo directly for CRUD operations
func GetMediaById(db *sqlx.DB, id int64) (*model.Media, error) {
	return mediaRepo.GetById(db, id)
}

// Deprecated: Use mediaRepo directly for CRUD operations
func GetMediasByUserId(db *sqlx.DB, userId uint64) ([]*model.Media, error) {
	return mediaRepo.GetByUserId(db, userId)
}

// Deprecated: Use mediaRepo directly for CRUD operations
func GetMediasByTweetId(db *sqlx.DB, tweetId int64) ([]*model.Media, error) {
	return mediaRepo.GetByTweetId(db, tweetId)
}

// Deprecated: Use mediaRepo directly for CRUD operations
func GetMediaByLocation(db *sqlx.DB, location string) (*model.Media, error) {
	return mediaRepo.GetByLocation(db, location)
}

// Deprecated: Use mediaRepo directly for CRUD operations
func UpdateMedia(db *sqlx.DB, media *model.Media) error {
	return mediaRepo.Update(db, media)
}

// Deprecated: Use mediaRepo directly for CRUD operations
func DeleteMedia(db *sqlx.DB, id int64) error {
	return mediaRepo.Delete(db, id)
}
