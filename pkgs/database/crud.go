package database

import (
	"time"

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

// User CRUD operations - backward compatibility wrappers
func CreateUser(db *sqlx.DB, usr *User) error {
	return userRepo.Create(db, usr)
}

func DelUser(db *sqlx.DB, uid uint64) error {
	return userRepo.Delete(db, uid)
}

func GetUserById(db *sqlx.DB, uid uint64) (*User, error) {
	return userRepo.GetById(db, uid)
}

func UpdateUser(db *sqlx.DB, usr *User) error {
	return userRepo.Update(db, usr)
}

// UserEntity CRUD operations - backward compatibility wrappers
func CreateUserEntity(db *sqlx.DB, entity *UserEntity) error {
	return userRepo.CreateEntity(db, entity)
}

func DelUserEntity(db *sqlx.DB, id uint32) error {
	return userRepo.DeleteEntity(db, id)
}

func GetUserEntity(db *sqlx.DB, uid uint64, parentDir string) (*UserEntity, error) {
	return userRepo.GetEntity(db, uid, parentDir)
}

func GetUserEntityById(db *sqlx.DB, id int) (*UserEntity, error) {
	return userRepo.GetEntityById(db, id)
}

func UpdateUserEntity(db *sqlx.DB, entity *UserEntity) error {
	return userRepo.UpdateEntity(db, entity)
}

func UpdateUserEntityMediCount(db *sqlx.DB, eid int, count int) error {
	return userRepo.UpdateEntityMediaCount(db, eid, count)
}

func UpdateUserEntityTweetStat(db *sqlx.DB, eid int, baseline time.Time, count int) error {
	return userRepo.UpdateEntityTweetStat(db, eid, baseline, count)
}

func SetUserEntityLatestReleaseTime(db *sqlx.DB, id int, t time.Time) error {
	return userRepo.SetEntityLatestReleaseTime(db, id, t)
}

func RecordUserPreviousName(db *sqlx.DB, uid uint64, name string, screenName string) error {
	return userRepo.RecordPreviousName(db, uid, name, screenName)
}

// UserLink CRUD operations - backward compatibility wrappers
func CreateUserLink(db *sqlx.DB, lnk *UserLink) error {
	return userRepo.CreateLink(db, lnk)
}

func DelUserLink(db *sqlx.DB, id int32) error {
	return userRepo.DeleteLink(db, id)
}

func GetUserLinks(db *sqlx.DB, uid uint64) ([]*UserLink, error) {
	return userRepo.GetLinks(db, uid)
}

func GetUserLink(db *sqlx.DB, uid uint64, parentLstEntityId int32) (*UserLink, error) {
	return userRepo.GetLink(db, uid, parentLstEntityId)
}

func UpdateUserLink(db *sqlx.DB, id int32, name string) error {
	return userRepo.UpdateLink(db, id, name)
}

// List CRUD operations - backward compatibility wrappers
func CreateLst(db *sqlx.DB, lst *Lst) error {
	return listRepo.Create(db, lst)
}

func DelLst(db *sqlx.DB, lid uint64) error {
	return listRepo.Delete(db, lid)
}

func GetLst(db *sqlx.DB, lid uint64) (*Lst, error) {
	return listRepo.GetById(db, lid)
}

func UpdateLst(db *sqlx.DB, lst *Lst) error {
	return listRepo.Update(db, lst)
}

// ListEntity CRUD operations - backward compatibility wrappers
func CreateLstEntity(db *sqlx.DB, entity *ListEntity) error {
	return listRepo.CreateEntity(db, entity)
}

func DelLstEntity(db *sqlx.DB, id int) error {
	return listRepo.DeleteEntity(db, id)
}

func GetListEntityById(db *sqlx.DB, id int) (*ListEntity, error) {
	return listRepo.GetEntityById(db, id)
}

func GetListEntity(db *sqlx.DB, lid int64, parentDir string) (*ListEntity, error) {
	return listRepo.GetEntity(db, lid, parentDir)
}

func UpdateLstEntity(db *sqlx.DB, entity *ListEntity) error {
	return listRepo.UpdateEntity(db, entity)
}

// Tweet CRUD operations - backward compatibility wrappers
func CreateTweet(db *sqlx.DB, tweet *Tweet) error {
	return tweetRepo.Create(db, tweet)
}

func GetTweetById(db *sqlx.DB, id int64) (*Tweet, error) {
	return tweetRepo.GetById(db, id)
}

func GetTweetsByUserId(db *sqlx.DB, userId uint64) ([]*Tweet, error) {
	return tweetRepo.GetByUserId(db, userId)
}

func UpdateTweet(db *sqlx.DB, tweet *Tweet) error {
	return tweetRepo.Update(db, tweet)
}

func DeleteTweet(db *sqlx.DB, id int64) error {
	return tweetRepo.Delete(db, id)
}

func GetTweetByTweetId(db *sqlx.DB, twitterId uint64) (*Tweet, error) {
	return tweetRepo.GetByTweetId(db, twitterId)
}

// Media CRUD operations - backward compatibility wrappers
func CreateMedia(db *sqlx.DB, media *Media) error {
	return mediaRepo.Create(db, media)
}

func GetMediaById(db *sqlx.DB, id int64) (*Media, error) {
	return mediaRepo.GetById(db, id)
}

func GetMediasByUserId(db *sqlx.DB, userId uint64) ([]*Media, error) {
	return mediaRepo.GetByUserId(db, userId)
}

func GetMediasByTweetId(db *sqlx.DB, tweetId int64) ([]*Media, error) {
	return mediaRepo.GetByTweetId(db, tweetId)
}

func GetMediaByLocation(db *sqlx.DB, location string) (*Media, error) {
	return mediaRepo.GetByLocation(db, location)
}

func UpdateMedia(db *sqlx.DB, media *Media) error {
	return mediaRepo.Update(db, media)
}

func DeleteMedia(db *sqlx.DB, id int64) error {
	return mediaRepo.Delete(db, id)
}

// Helper functions for tweet-media relationships
func GetTweetsWithMedia(db *sqlx.DB, userId uint64) ([]map[string]interface{}, error) {
	return tweetRepo.GetWithMedia(db, userId)
}
