package downloading

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/WangWilly/xSync/internal/database"
	"github.com/jmoiron/sqlx"
)

// 路径Plus
type SmartPath interface {
	Path() (string, error)
	Create(name string) error
	Rename(string) error
	Remove() error
	Name() string
	Id() int
	Recorded() bool
}

////////////////////////////////////////////////////////////////////////////////

func syncPath(path SmartPath, expectedName string) error {
	if !path.Recorded() {
		return path.Create(expectedName)
	}

	if path.Name() != expectedName {
		return path.Rename(expectedName)
	}

	p, err := path.Path()
	if err != nil {
		return err
	}

	return os.MkdirAll(p, 0755)
}

////////////////////////////////////////////////////////////////////////////////

type UserSmartPath struct {
	record  *database.UserEntity
	db      *sqlx.DB
	created bool
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

	return &UserSmartPath{record: userEntity, db: db, created: created}, nil
}

func (user *UserSmartPath) Create(name string) error {
	user.record.Name = name
	path, _ := user.Path()
	if err := os.MkdirAll(path, 0755); err != nil {
		return err
	}

	if err := database.CreateUserEntity(user.db, user.record); err != nil {
		return err
	}
	user.created = true
	return nil
}

func (user *UserSmartPath) Remove() error {
	path, _ := user.Path()

	if err := os.RemoveAll(path); err != nil {
		return err
	}
	if err := database.DelUserEntity(user.db, uint32(user.record.Id.Int32)); err != nil {
		return err
	}
	user.created = false
	return nil
}

func (user *UserSmartPath) Rename(title string) error {
	if !user.created {
		return fmt.Errorf("user entity [%s:%d] was not created", user.record.ParentDir, user.record.Uid)
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

	user.record.Name = title
	return database.UpdateUserEntity(user.db, user.record)
}

func (user *UserSmartPath) Path() (string, error) {
	return user.record.Path(), nil
}

func (user *UserSmartPath) Name() string {
	if user.record.Name == "" {
		panic(fmt.Errorf("the name of user entity [%s:%d] was unset", user.record.ParentDir, user.record.Uid))
	}
	return user.record.Name
}

func (user *UserSmartPath) Id() int {
	if !user.created {
		panic(fmt.Sprintf("user entity [%s:%d] was not created", user.record.ParentDir, user.record.Uid))
	}
	return int(user.record.Id.Int32)
}

func (user *UserSmartPath) LatestReleaseTime() time.Time {
	if !user.created {
		panic(fmt.Sprintf("user entity [%s:%d] was not created", user.record.ParentDir, user.record.Uid))
	}
	return user.record.LatestReleaseTime.Time
}

func (user *UserSmartPath) SetLatestReleaseTime(t time.Time) error {
	if !user.created {
		return fmt.Errorf("user entity [%s:%d] was not created", user.record.ParentDir, user.record.Uid)
	}
	err := database.SetUserEntityLatestReleaseTime(user.db, int(user.record.Id.Int32), t)
	if err == nil {
		user.record.LatestReleaseTime.Scan(t)
	}
	return err
}

func (user *UserSmartPath) Uid() uint64 {
	return user.record.Uid
}

func (user *UserSmartPath) Recorded() bool {
	return user.created
}

////////////////////////////////////////////////////////////////////////////////

type ListSmartPath struct {
	record  *database.ListEntity
	db      *sqlx.DB
	created bool
}

func NewListSmartPath(db *sqlx.DB, lid int64, parentDir string) (*ListSmartPath, error) {
	created := true
	record, err := database.GetListEntity(db, lid, parentDir)
	if err != nil {
		return nil, err
	}
	if record == nil {
		record = &database.ListEntity{}
		record.LstId = lid
		record.ParentDir = parentDir
		created = false
	}
	return &ListSmartPath{record: record, db: db, created: created}, nil
}

func (le *ListSmartPath) Create(name string) error {
	le.record.Name = name
	path, _ := le.Path()
	if err := os.MkdirAll(path, 0755); err != nil {
		return nil
	}

	if err := database.CreateLstEntity(le.db, le.record); err != nil {
		return err
	}
	le.created = true
	return nil
}

func (le *ListSmartPath) Remove() error {
	if !le.created {
		return fmt.Errorf("list entity [%s:%d] was not created", le.record.ParentDir, le.record.LstId)
	}

	path, _ := le.Path()
	if err := os.RemoveAll(path); err != nil {
		return err
	}
	if err := database.DelLstEntity(le.db, int(le.record.Id.Int32)); err != nil {
		return err
	}
	le.created = false
	return nil
}

func (le *ListSmartPath) Rename(title string) error {
	if !le.created {
		return fmt.Errorf("list entity [%s:%d] was not created", le.record.ParentDir, le.record.LstId)
	}

	path, _ := le.Path()
	newPath := filepath.Join(filepath.Dir(path), title)
	err := os.Rename(path, newPath)
	if os.IsNotExist(err) {
		err = os.Mkdir(newPath, 0755)
	}
	if err != nil && !os.IsExist(err) {
		return err
	}

	le.record.Name = title
	return database.UpdateLstEntity(le.db, le.record)
}

func (le *ListSmartPath) Path() (string, error) {
	return le.record.Path(), nil
}

func (le ListSmartPath) Name() string {
	if le.record.Name == "" {
		panic(fmt.Sprintf("the name of list entity [%s:%d] was unset", le.record.ParentDir, le.record.LstId))
	}

	return le.record.Name
}

func (le *ListSmartPath) Id() int {
	if !le.created {
		panic(fmt.Sprintf("list entity [%s:%d] was not created", le.record.ParentDir, le.record.LstId))
	}

	return int(le.record.Id.Int32)
}

func (le *ListSmartPath) Recorded() bool {
	return le.created
}

////////////////////////////////////////////////////////////////////////////////

func updateUserLink(lnk *database.UserLink, db *sqlx.DB, path string) error {
	name := filepath.Base(path)

	linkpath, err := lnk.Path(db)
	if err != nil {
		return err
	}
	path, err = filepath.Abs(path)
	if err != nil {
		return err
	}

	if lnk.Name == name {
		// 用户未改名，但仍应确保链接存在
		err = os.Symlink(path, linkpath)
		if os.IsExist(err) {
			err = nil
		}
		return err
	}

	newlinkpath := filepath.Join(filepath.Dir(linkpath), name)

	if err = os.RemoveAll(linkpath); err != nil {
		return err
	}
	if err = os.Symlink(path, newlinkpath); err != nil && !os.IsExist(err) {
		return err
	}

	if err = database.UpdateUserLink(db, lnk.Id.Int32, name); err != nil {
		return err
	}

	lnk.Name = name
	return nil
}
