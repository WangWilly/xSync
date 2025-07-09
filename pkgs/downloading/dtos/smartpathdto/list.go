package smartpathdto

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/WangWilly/xSync/pkgs/database"
	"github.com/jmoiron/sqlx"
)

type ListSmartPath struct {
	record  *database.ListEntity
	db      *sqlx.DB
	Created bool
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
	return &ListSmartPath{record: record, db: db, Created: created}, nil
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
	le.Created = true
	return nil
}

func (le *ListSmartPath) Remove() error {
	if !le.Created {
		return fmt.Errorf("list entity [%s:%d] was not created", le.record.ParentDir, le.record.LstId)
	}

	path, _ := le.Path()
	if err := os.RemoveAll(path); err != nil {
		return err
	}
	if err := database.DelLstEntity(le.db, int(le.record.Id.Int32)); err != nil {
		return err
	}
	le.Created = false
	return nil
}

func (le *ListSmartPath) Rename(title string) error {
	if !le.Created {
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
	if !le.Created {
		panic(fmt.Sprintf("list entity [%s:%d] was not created", le.record.ParentDir, le.record.LstId))
	}

	return int(le.record.Id.Int32)
}

func (le *ListSmartPath) IsSyncToDb() bool {
	return le.Created
}
