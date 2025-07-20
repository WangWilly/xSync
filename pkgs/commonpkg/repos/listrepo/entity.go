package listrepo

import (
	"database/sql"
	"fmt"
	"path/filepath"

	"github.com/WangWilly/xSync/pkgs/commonpkg/model"
	"github.com/jmoiron/sqlx"
)

func (r *Repo) CreateEntity(db *sqlx.DB, entity *model.ListEntity) error {
	abs, err := filepath.Abs(entity.ParentDir)
	if err != nil {
		return err
	}
	entity.ParentDir = abs

	stmt := `INSERT INTO lst_entities(lst_id, name, parent_dir, folder_name, storage_saved)
			VALUES(:lst_id, :name, :parent_dir, :folder_name, :storage_saved)
			RETURNING id, lst_id, name, parent_dir, folder_name, storage_saved, created_at, updated_at`
	rows, err := db.NamedQuery(stmt, entity)
	if err != nil {
		return err
	}
	defer rows.Close()

	if !rows.Next() {
		return fmt.Errorf("no rows returned for entity with lst_id %d and parent_dir %s", entity.LstId, entity.ParentDir)
	}
	if err := rows.StructScan(entity); err != nil {
		return err
	}
	return nil
}

func (r *Repo) UpsertEntity(db *sqlx.DB, entity *model.ListEntity) error {
	abs, err := filepath.Abs(entity.ParentDir)
	if err != nil {
		return err
	}
	entity.ParentDir = abs

	stmt := `INSERT INTO lst_entities(lst_id, name, parent_dir, folder_name, storage_saved)
		VALUES(:lst_id, :name, :parent_dir, :folder_name, :storage_saved)
		ON CONFLICT(lst_id) DO UPDATE SET name=:name, parent_dir=:parent_dir, folder_name=:folder_name, storage_saved=:storage_saved, updated_at=CURRENT_TIMESTAMP
		RETURNING id, lst_id, name, parent_dir, folder_name, storage_saved, created_at, updated_at`
	rows, err := db.NamedQuery(stmt, entity)
	if err != nil {
		return err
	}
	defer rows.Close()

	if !rows.Next() {
		return fmt.Errorf("no rows returned for upsert of entity with lst_id %d and parent_dir %s", entity.LstId, entity.ParentDir)
	}
	if err := rows.StructScan(entity); err != nil {
		return err
	}
	return nil
}

////////////////////////////////////////////////////////////////////////////////

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

////////////////////////////////////////////////////////////////////////////////

func (r *Repo) UpdateEntity(db *sqlx.DB, entity *model.ListEntity) error {
	stmt := `UPDATE lst_entities SET name=?, parent_dir=?, folder_name=?, storage_saved=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`
	_, err := db.Exec(stmt, entity.Name, entity.ParentDir, entity.FolderName, entity.StorageSaved, entity.Id)
	return err
}

func (r *Repo) UpdateEntityStorageSavedByTwitterId(db *sqlx.DB, twitterId uint64, saved bool) error {
	stmt := `UPDATE lst_entities SET storage_saved=?, updated_at=CURRENT_TIMESTAMP WHERE lst_id=?`
	_, err := db.Exec(stmt, saved, twitterId)
	return err
}

////////////////////////////////////////////////////////////////////////////////

func (r *Repo) DeleteEntity(db *sqlx.DB, id int) error {
	stmt := `DELETE FROM lst_entities WHERE id=?`
	_, err := db.Exec(stmt, id)
	return err
}
