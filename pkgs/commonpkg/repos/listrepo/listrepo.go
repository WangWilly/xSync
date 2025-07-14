package listrepo

import (
	"database/sql"
	"path/filepath"

	"github.com/WangWilly/xSync/pkgs/commonpkg/model"
	"github.com/jmoiron/sqlx"
)

type Repo struct{}

func New() *Repo {
	return &Repo{}
}

func (r *Repo) Create(db *sqlx.DB, lst *model.List) error {
	stmt := `INSERT INTO lsts(id, name, owner_uid) VALUES(:id, :name, :owner_uid)`
	_, err := db.NamedExec(stmt, &lst)
	return err
}

func (r *Repo) Delete(db *sqlx.DB, lid uint64) error {
	stmt := `DELETE FROM lsts WHERE id=?`
	_, err := db.Exec(stmt, lid)
	return err
}

func (r *Repo) GetById(db *sqlx.DB, lid uint64) (*model.List, error) {
	stmt := `SELECT * FROM lsts WHERE id = ?`
	result := &model.List{}
	err := db.Get(result, stmt, lid)
	if err == sql.ErrNoRows {
		err = nil
		result = nil
	}
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (r *Repo) Update(db *sqlx.DB, lst *model.List) error {
	stmt := `UPDATE lsts SET name=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`
	_, err := db.Exec(stmt, lst.Name, lst.Id)
	return err
}

func (r *Repo) CreateEntity(db *sqlx.DB, entity *model.ListEntity) error {
	abs, err := filepath.Abs(entity.ParentDir)
	if err != nil {
		return err
	}
	entity.ParentDir = abs

	stmt := `INSERT INTO lst_entities(id, lst_id, name, parent_dir) VALUES(:id, :lst_id, :name, :parent_dir)`
	result, err := db.NamedExec(stmt, &entity)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	entity.Id.Scan(id)
	return nil
}

func (r *Repo) DeleteEntity(db *sqlx.DB, id int) error {
	stmt := `DELETE FROM lst_entities WHERE id=?`
	_, err := db.Exec(stmt, id)
	return err
}

func (r *Repo) GetEntityById(db *sqlx.DB, id int) (*model.ListEntity, error) {
	stmt := `SELECT * FROM lst_entities WHERE id=?`
	result := &model.ListEntity{}
	err := db.Get(result, stmt, id)
	if err == sql.ErrNoRows {
		err = nil
		result = nil
	}
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (r *Repo) GetEntity(db *sqlx.DB, lid int64, parentDir string) (*model.ListEntity, error) {
	parentDir, err := filepath.Abs(parentDir)
	if err != nil {
		return nil, err
	}

	stmt := `SELECT * FROM lst_entities WHERE lst_id=? AND parent_dir=?`
	result := &model.ListEntity{}
	err = db.Get(result, stmt, lid, parentDir)
	if err == sql.ErrNoRows {
		err = nil
		result = nil
	}
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (r *Repo) UpdateEntity(db *sqlx.DB, entity *model.ListEntity) error {
	stmt := `UPDATE lst_entities SET name=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`
	_, err := db.Exec(stmt, entity.Name, entity.Id.Int32)
	return err
}
