package smartpathdto

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/WangWilly/xSync/internal/database"
	"github.com/jmoiron/sqlx"
)

type UserSmartPath struct {
	Record  *database.UserEntity
	db      *sqlx.DB
	Created bool
}

func NewUserSmartPath(db *sqlx.DB, uid uint64, parentDir string) (*UserSmartPath, error) {
	created := true
	userEntity, err := database.GetUserEntity(db, uid, parentDir)
	if err != nil {
		return nil, err
	}

	if userEntity == nil {
		userEntity = &database.UserEntity{}
		userEntity.Uid = uid
		userEntity.ParentDir = parentDir
		created = false
	}

	return &UserSmartPath{Record: userEntity, db: db, Created: created}, nil
}

func RebuildUserSmartPath(db *sqlx.DB, record *database.UserEntity) (*UserSmartPath, error) {
	if record == nil {
		return nil, fmt.Errorf("record is nil")
	}
	if record.Uid == 0 {
		return nil, fmt.Errorf("record uid is zero")
	}

	return &UserSmartPath{
		Record:  record,
		db:      db,
		Created: true,
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
	user.Created = true
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
	user.Created = false
	return nil
}

func (user *UserSmartPath) Rename(title string) error {
	if !user.Created {
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
	if !user.Created {
		panic(fmt.Sprintf("user entity [%s:%d] was not created", user.Record.ParentDir, user.Record.Uid))
	}
	return int(user.Record.Id.Int32)
}

func (user *UserSmartPath) LatestReleaseTime() time.Time {
	if !user.Created {
		panic(fmt.Sprintf("user entity [%s:%d] was not created", user.Record.ParentDir, user.Record.Uid))
	}
	return user.Record.LatestReleaseTime.Time
}

func (user *UserSmartPath) SetLatestReleaseTime(t time.Time) error {
	if !user.Created {
		return fmt.Errorf("user entity [%s:%d] was not created", user.Record.ParentDir, user.Record.Uid)
	}
	err := database.SetUserEntityLatestReleaseTime(user.db, int(user.Record.Id.Int32), t)
	if err == nil {
		user.Record.LatestReleaseTime.Scan(t)
	}
	return err
}

func (user *UserSmartPath) Uid() uint64 {
	return user.Record.Uid
}

func (user *UserSmartPath) Recorded() bool {
	return user.Created
}
