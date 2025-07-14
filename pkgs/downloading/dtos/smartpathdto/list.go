package smartpathdto

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/WangWilly/xSync/pkgs/clipkg/database"
	"github.com/WangWilly/xSync/pkgs/commonpkg/model"
	"github.com/jmoiron/sqlx"
)

type ListSmartPath struct {
	record  *model.ListEntity
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
		record = &model.ListEntity{}
		record.LstId = lid
		record.ParentDir = parentDir
		created = false
	}
	return &ListSmartPath{record: record, db: db, Created: created}, nil
}

func (listsp *ListSmartPath) Create(name string) error {
	listsp.record.Name = name
	path, _ := listsp.Path()
	if err := os.MkdirAll(path, 0755); err != nil {
		return nil
	}

	if err := database.CreateLstEntity(listsp.db, listsp.record); err != nil {
		return err
	}
	listsp.Created = true
	return nil
}

func (listsp *ListSmartPath) Remove() error {
	if !listsp.Created {
		return fmt.Errorf("list entity [%s:%d] was not created", listsp.record.ParentDir, listsp.record.LstId)
	}

	path, _ := listsp.Path()
	if err := os.RemoveAll(path); err != nil {
		return err
	}
	if err := database.DelLstEntity(listsp.db, int(listsp.record.Id.Int32)); err != nil {
		return err
	}
	listsp.Created = false
	return nil
}

func (listsp *ListSmartPath) Rename(title string) error {
	if !listsp.Created {
		return fmt.Errorf("list entity [%s:%d] was not created", listsp.record.ParentDir, listsp.record.LstId)
	}

	path, _ := listsp.Path()
	newPath := filepath.Join(filepath.Dir(path), title)
	err := os.Rename(path, newPath)
	if os.IsNotExist(err) {
		err = os.Mkdir(newPath, 0755)
	}
	if err != nil && !os.IsExist(err) {
		return err
	}

	listsp.record.Name = title
	return database.UpdateLstEntity(listsp.db, listsp.record)
}

func (listsp *ListSmartPath) Path() (string, error) {
	return listsp.record.Path(), nil
}

func (listsp ListSmartPath) Name() string {
	if listsp.record.Name == "" {
		panic(fmt.Sprintf("the name of list entity [%s:%d] was unset", listsp.record.ParentDir, listsp.record.LstId))
	}

	return listsp.record.Name
}

func (listsp *ListSmartPath) Id() int {
	if !listsp.Created {
		panic(fmt.Sprintf("list entity [%s:%d] was not created", listsp.record.ParentDir, listsp.record.LstId))
	}

	return int(listsp.record.Id.Int32)
}

func (listsp *ListSmartPath) IsSyncToDb() bool {
	return listsp.Created
}
