package smartpathdto

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/WangWilly/xSync/pkgs/clipkg/database"
	"github.com/WangWilly/xSync/pkgs/commonpkg/model"
	"github.com/jmoiron/sqlx"
)

type UserSmartPath struct {
	Record     *model.UserEntity
	db         *sqlx.DB
	isSyncToDb bool

	Depth int
}

func New(record *model.UserEntity, depth int) *UserSmartPath {
	if record == nil {
		return nil
	}
	return &UserSmartPath{
		Record:     record,
		db:         nil,
		isSyncToDb: true,
		Depth:      depth,
	}
}

func NewUserSmartPath(db *sqlx.DB, twitterId uint64, parentDir string) (*UserSmartPath, error) {
	created := true
	userEntity, err := database.GetUserEntity(db, twitterId, parentDir)
	if err != nil {
		return nil, err
	}

	if userEntity == nil {
		userEntity = &model.UserEntity{}
		userEntity.Uid = twitterId
		userEntity.ParentDir = parentDir
		created = false
	}

	return &UserSmartPath{Record: userEntity, db: db, isSyncToDb: created}, nil
}

func RebuildUserSmartPath(db *sqlx.DB, record *model.UserEntity) (*UserSmartPath, error) {
	if record == nil {
		return nil, fmt.Errorf("record is nil")
	}
	if record.Uid == 0 {
		return nil, fmt.Errorf("record uid is zero")
	}

	return &UserSmartPath{
		Record:     record,
		db:         db,
		isSyncToDb: true,
	}, nil
}

func (user *UserSmartPath) Create(name string) error {
	user.Record.Name = name
	path, _ := user.Path()
	if err := os.MkdirAll(path, 0755); err != nil {
		return err
	}

	if err := database.CreateUserEntity(user.db, user.Record); err != nil {
		return err
	}
	user.isSyncToDb = true
	return nil
}

func (user *UserSmartPath) Remove() error {
	path, _ := user.Path()

	if err := os.RemoveAll(path); err != nil {
		return err
	}
	if err := database.DelUserEntity(user.db, uint32(user.Record.Id.Int32)); err != nil {
		return err
	}
	user.isSyncToDb = false
	return nil
}

func (user *UserSmartPath) Rename(title string) error {
	if !user.isSyncToDb {
		return fmt.Errorf("user entity [%s:%d] was not created", user.Record.ParentDir, user.Record.Uid)
	}

	old, _ := user.Path()
	newPath := filepath.Join(filepath.Dir(old), title)

	err := os.Rename(old, newPath)
	if os.IsNotExist(err) {
		err = os.Mkdir(newPath, 0755)
	}
	if err != nil && !os.IsExist(err) {
		return err
	}

	user.Record.Name = title
	return database.UpdateUserEntity(user.db, user.Record)
}

func (user *UserSmartPath) Path() (string, error) {
	return user.Record.Path(), nil
}

func (user *UserSmartPath) Name() string {
	if user.Record.Name == "" {
		panic(fmt.Errorf("the name of user entity [%s:%d] was unset", user.Record.ParentDir, user.Record.Uid))
	}
	return user.Record.Name
}

func (user *UserSmartPath) Id() int {
	if !user.isSyncToDb {
		panic(fmt.Sprintf("user entity [%s:%d] was not created", user.Record.ParentDir, user.Record.Uid))
	}
	return int(user.Record.Id.Int32)
}

func (user *UserSmartPath) LatestReleaseTime() time.Time {
	if !user.isSyncToDb {
		panic(fmt.Sprintf("user entity [%s:%d] was not created", user.Record.ParentDir, user.Record.Uid))
	}
	return user.Record.LatestReleaseTime.Time
}

func (user *UserSmartPath) SetLatestReleaseTime(t time.Time) error {
	if !user.isSyncToDb {
		return fmt.Errorf("user entity [%s:%d] was not created", user.Record.ParentDir, user.Record.Uid)
	}
	err := database.SetUserEntityLatestReleaseTime(user.db, int(user.Record.Id.Int32), t)
	if err == nil {
		user.Record.LatestReleaseTime.Scan(t)
	}
	return err
}

func (user *UserSmartPath) TwitterId() uint64 {
	return user.Record.Uid
}

func (user *UserSmartPath) IsSyncToDb() bool {
	return user.isSyncToDb
}
