package database

import (
	"context"
	"time"

	"github.com/WangWilly/xSync/pkgs/commonpkg/model"
	"github.com/WangWilly/xSync/pkgs/commonpkg/repos/listrepo"
	"github.com/WangWilly/xSync/pkgs/commonpkg/repos/mediarepo"
	"github.com/WangWilly/xSync/pkgs/commonpkg/repos/tweetrepo"
	"github.com/WangWilly/xSync/pkgs/commonpkg/repos/userrepo"
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
	ctx := context.Background()
	return userRepo.Create(ctx, db, usr)
}

// Deprecated: Use userRepo directly for CRUD operations
func DelUser(db *sqlx.DB, uid uint64) error {
	ctx := context.Background()
	return userRepo.Delete(ctx, db, uid)
}

// Deprecated: Use userRepo directly for CRUD operations
func GetUserById(db *sqlx.DB, uid uint64) (*model.User, error) {
	ctx := context.Background()
	return userRepo.GetById(ctx, db, uid)
}

// Deprecated: Use userRepo directly for CRUD operations
func UpdateUser(db *sqlx.DB, usr *model.User) error {
	ctx := context.Background()
	return userRepo.Update(ctx, db, usr)
}

// Deprecated: Use userRepo directly for CRUD operations
//
// UserEntity CRUD operations - backward compatibility wrappers
func CreateUserEntity(db *sqlx.DB, entity *model.UserEntity) error {
	ctx := context.Background()
	return userRepo.CreateEntity(ctx, db, entity)
}

// Deprecated: Use userRepo directly for CRUD operations
func DelUserEntity(db *sqlx.DB, id uint32) error {
	ctx := context.Background()
	return userRepo.DeleteEntity(ctx, db, id)
}

// Deprecated: Use userRepo directly for CRUD operations
func GetUserEntity(db *sqlx.DB, uid uint64, parentDir string) (*model.UserEntity, error) {
	ctx := context.Background()
	return userRepo.GetEntity(ctx, db, uid, parentDir)
}

// Deprecated: Use userRepo directly for CRUD operations
func GetUserEntityById(db *sqlx.DB, id int) (*model.UserEntity, error) {
	ctx := context.Background()
	return userRepo.GetEntityById(ctx, db, id)
}

// Deprecated: Use userRepo directly for CRUD operations
func UpdateUserEntity(db *sqlx.DB, entity *model.UserEntity) error {
	ctx := context.Background()
	return userRepo.UpdateEntity(ctx, db, entity)
}

// Deprecated: Use userRepo directly for CRUD operations
func UpdateUserEntityMediCount(db *sqlx.DB, eid int, count int) error {
	ctx := context.Background()
	return userRepo.UpdateEntityMediaCount(ctx, db, eid, count)
}

// Deprecated: Use userRepo directly for CRUD operations
func UpdateUserEntityTweetStat(db *sqlx.DB, eid int, baseline time.Time, count int) error {
	ctx := context.Background()
	return userRepo.UpdateEntityTweetStat(ctx, db, eid, baseline, count)
}

// Deprecated: Use userRepo directly for CRUD operations
func SetUserEntityLatestReleaseTime(db *sqlx.DB, id int, t time.Time) error {
	ctx := context.Background()
	return userRepo.SetEntityLatestReleaseTime(ctx, db, id, t)
}

// Deprecated: Use userRepo directly for CRUD operations
func CreateUserPreviousName(db *sqlx.DB, uid uint64, name string, screenName string) error {
	ctx := context.Background()
	return userRepo.CreatePreviousName(ctx, db, uid, name, screenName)
}

// Deprecated: Use userRepo directly for CRUD operations
// model.UserLink CRUD operations - backward compatibility wrappers
func CreateUserLink(db *sqlx.DB, lnk *model.UserLink) error {
	ctx := context.Background()
	return userRepo.CreateLink(ctx, db, lnk)
}

// Deprecated: Use userRepo directly for CRUD operations
func DelUserLink(db *sqlx.DB, id int32) error {
	ctx := context.Background()
	return userRepo.DeleteLink(ctx, db, id)
}

// Deprecated: Use userRepo directly for CRUD operations
func GetUserLinks(db *sqlx.DB, uid uint64) ([]*model.UserLink, error) {
	ctx := context.Background()
	return userRepo.GetLinks(ctx, db, uid)
}

// Deprecated: Use userRepo directly for CRUD operations
func GetUserLink(db *sqlx.DB, uid uint64, parentLstEntityId int32) (*model.UserLink, error) {
	ctx := context.Background()
	return userRepo.GetLink(ctx, db, uid, parentLstEntityId)
}

// Deprecated: Use userRepo directly for CRUD operations
func UpdateUserLink(db *sqlx.DB, id int32, name string) error {
	ctx := context.Background()
	return userRepo.UpdateLink(ctx, db, id, name)
}

// Deprecated: Use listRepo directly for CRUD operations
//
// List CRUD operations - backward compatibility wrappers
func CreateLst(db *sqlx.DB, lst *model.List) error {
	ctx := context.Background()
	return listRepo.Create(ctx, db, lst)
}

// Deprecated: Use listRepo directly for CRUD operations
func DelLst(db *sqlx.DB, lid uint64) error {
	ctx := context.Background()
	return listRepo.Delete(ctx, db, lid)
}

// Deprecated: Use listRepo directly for CRUD operations
func GetLst(db *sqlx.DB, lid uint64) (*model.List, error) {
	ctx := context.Background()
	return listRepo.GetById(ctx, db, lid)
}

// Deprecated: Use listRepo directly for CRUD operations
func UpdateLst(db *sqlx.DB, lst *model.List) error {
	ctx := context.Background()
	return listRepo.Update(ctx, db, lst)
}

// Deprecated: Use listRepo directly for CRUD operations
// model.ListEntity CRUD operations - backward compatibility wrappers
func CreateLstEntity(db *sqlx.DB, entity *model.ListEntity) error {
	ctx := context.Background()
	return listRepo.CreateEntity(ctx, db, entity)
}

// Deprecated: Use listRepo directly for CRUD operations
func DelLstEntity(db *sqlx.DB, id int) error {
	ctx := context.Background()
	return listRepo.DeleteEntity(ctx, db, id)
}

// Deprecated: Use listRepo directly for CRUD operations
func GetListEntityById(db *sqlx.DB, id int) (*model.ListEntity, error) {
	ctx := context.Background()
	return listRepo.GetEntityById(ctx, db, id)
}

// Deprecated: Use listRepo directly for CRUD operations
func GetListEntity(db *sqlx.DB, lid int64, parentDir string) (*model.ListEntity, error) {
	ctx := context.Background()
	return listRepo.GetEntity(ctx, db, lid, parentDir)
}

// Deprecated: Use listRepo directly for CRUD operations
func UpdateLstEntity(db *sqlx.DB, entity *model.ListEntity) error {
	ctx := context.Background()
	return listRepo.UpdateEntity(ctx, db, entity)
}

// Deprecated: Use tweetRepo directly for CRUD operations
//
// model.Tweet CRUD operations - backward compatibility wrappers
func CreateTweet(db *sqlx.DB, tweet *model.Tweet) error {
	ctx := context.Background()
	return tweetRepo.Create(ctx, db, tweet)
}

// Deprecated: Use tweetRepo directly for CRUD operations
func GetTweetById(db *sqlx.DB, id int64) (*model.Tweet, error) {
	ctx := context.Background()
	return tweetRepo.GetById(ctx, db, id)
}

// Deprecated: Use tweetRepo directly for CRUD operations
func GetTweetsByUserId(db *sqlx.DB, userId uint64) ([]*model.Tweet, error) {
	ctx := context.Background()
	return tweetRepo.GetByUserId(ctx, db, userId)
}

// Deprecated: Use tweetRepo directly for CRUD operations
func UpdateTweet(db *sqlx.DB, tweet *model.Tweet) error {
	ctx := context.Background()
	return tweetRepo.Update(ctx, db, tweet)
}

// Deprecated: Use tweetRepo directly for CRUD operations
func DeleteTweet(db *sqlx.DB, id int64) error {
	ctx := context.Background()
	return tweetRepo.Delete(ctx, db, id)
}

// Deprecated: Use tweetRepo directly for CRUD operations
func GetTweetByTweetId(db *sqlx.DB, twitterId uint64) (*model.Tweet, error) {
	ctx := context.Background()
	return tweetRepo.GetByTweetId(ctx, db, twitterId)
}

// Deprecated: Use tweetRepo directly for CRUD operations
//
// Helper functions for tweet-media relationships
func GetTweetsWithMedia(db *sqlx.DB, userId uint64) ([]map[string]interface{}, error) {
	ctx := context.Background()
	return tweetRepo.GetWithMedia(ctx, db, userId)
}

// Deprecated: Use mediaRepo directly for CRUD operations
//
// model.Media CRUD operations - backward compatibility wrappers
func CreateMedia(db *sqlx.DB, media *model.Media) error {
	ctx := context.Background()
	return mediaRepo.Create(ctx, db, media)
}

// Deprecated: Use mediaRepo directly for CRUD operations
func GetMediaById(db *sqlx.DB, id int64) (*model.Media, error) {
	ctx := context.Background()
	return mediaRepo.GetById(ctx, db, id)
}

// Deprecated: Use mediaRepo directly for CRUD operations
func GetMediasByUserId(db *sqlx.DB, userId uint64) ([]*model.Media, error) {
	ctx := context.Background()
	return mediaRepo.GetByUserId(ctx, db, userId)
}

// Deprecated: Use mediaRepo directly for CRUD operations
func GetMediasByTweetId(db *sqlx.DB, tweetId int64) ([]*model.Media, error) {
	ctx := context.Background()
	return mediaRepo.GetByTweetId(ctx, db, tweetId)
}

// Deprecated: Use mediaRepo directly for CRUD operations
func GetMediaByLocation(db *sqlx.DB, location string) (*model.Media, error) {
	ctx := context.Background()
	return mediaRepo.GetByLocation(ctx, db, location)
}

// Deprecated: Use mediaRepo directly for CRUD operations
func UpdateMedia(db *sqlx.DB, media *model.Media) error {
	ctx := context.Background()
	return mediaRepo.Update(ctx, db, media)
}

// Deprecated: Use mediaRepo directly for CRUD operations
func DeleteMedia(db *sqlx.DB, id int64) error {
	ctx := context.Background()
	return mediaRepo.Delete(ctx, db, id)
}
