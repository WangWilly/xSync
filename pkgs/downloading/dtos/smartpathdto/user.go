package smartpathdto

import (
	"fmt"
	"time"

	"github.com/WangWilly/xSync/pkgs/commonpkg/model"
)

type UserSmartPath struct {
	Record     *model.UserEntity
	isSyncToDb bool

	Depth int
}

func New(record *model.UserEntity, depth int) *UserSmartPath {
	if record == nil {
		return nil
	}
	return &UserSmartPath{
		Record:     record,
		isSyncToDb: true,
		Depth:      depth,
	}
}

func NewWithoutDepth(record *model.UserEntity) (*UserSmartPath, error) {
	if record == nil {
		return nil, fmt.Errorf("record is nil")
	}
	if record.Uid == 0 {
		return nil, fmt.Errorf("record uid is zero")
	}

	return &UserSmartPath{
		Record:     record,
		isSyncToDb: true,
	}, nil
}

////////////////////////////////////////////////////////////////////////////////

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
	return int(user.Record.Id)
}

func (user *UserSmartPath) LatestReleaseTime() time.Time {
	if !user.isSyncToDb {
		panic(fmt.Sprintf("user entity [%s:%d] was not created", user.Record.ParentDir, user.Record.Uid))
	}
	return user.Record.LatestReleaseTime.Time
}

func (user *UserSmartPath) TwitterId() uint64 {
	return user.Record.Uid
}

func (user *UserSmartPath) IsSyncToDb() bool {
	return user.isSyncToDb
}
