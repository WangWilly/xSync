package listentityrepo

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"

	"github.com/WangWilly/xSync/pkgs/commonpkg/model"
	"github.com/jmoiron/sqlx"
)

type repo struct{}

func New() *repo {
	return &repo{}
}

////////////////////////////////////////////////////////////////////////////////

func (r *repo) Create(ctx context.Context, db *sqlx.DB, entity *model.ListEntity) error {
	abs, err := filepath.Abs(entity.ParentDir)
	if err != nil {
		return err
	}
	entity.ParentDir = abs

	stmt := `INSERT INTO lst_entities(lst_id, name, parent_dir, folder_name, storage_saved)
			VALUES(:lst_id, :name, :parent_dir, :folder_name, :storage_saved)
			RETURNING id, lst_id, name, parent_dir, folder_name, storage_saved, created_at, updated_at`
	rows, err := db.NamedQueryContext(ctx, stmt, entity)
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

func (r *repo) Upsert(ctx context.Context, db *sqlx.DB, entity *model.ListEntity) error {
	abs, err := filepath.Abs(entity.ParentDir)
	if err != nil {
		return err
	}
	entity.ParentDir = abs

	stmt := `INSERT INTO lst_entities(lst_id, name, parent_dir, folder_name, storage_saved)
		VALUES(:lst_id, :name, :parent_dir, :folder_name, :storage_saved)
		ON CONFLICT(lst_id) DO UPDATE SET name=:name, parent_dir=:parent_dir, folder_name=:folder_name, storage_saved=:storage_saved, updated_at=CURRENT_TIMESTAMP
		RETURNING id, lst_id, name, parent_dir, folder_name, storage_saved, created_at, updated_at`
	rows, err := db.NamedQueryContext(ctx, stmt, entity)
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

func (r *repo) GetById(ctx context.Context, db *sqlx.DB, id int) (*model.ListEntity, error) {
	stmt := `SELECT * FROM lst_entities WHERE id=?`
	result := &model.ListEntity{}
	err := db.GetContext(ctx, result, stmt, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (r *repo) Get(ctx context.Context, db *sqlx.DB, lid int64, parentDir string) (*model.ListEntity, error) {
	parentDir, err := filepath.Abs(parentDir)
	if err != nil {
		return nil, err
	}

	stmt := `SELECT * FROM lst_entities WHERE lst_id=? AND parent_dir=?`
	result := &model.ListEntity{}
	err = db.GetContext(ctx, result, stmt, lid, parentDir)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return result, nil
}

////////////////////////////////////////////////////////////////////////////////

func (r *repo) Update(ctx context.Context, db *sqlx.DB, entity *model.ListEntity) error {
	stmt := `UPDATE lst_entities
			 SET
			 	name=:name,
				parent_dir=:parent_dir,
				folder_name=:folder_name,
				storage_saved=:storage_saved,
				updated_at=CURRENT_TIMESTAMP
			 WHERE id=?
			 RETURNING id, lst_id, name, parent_dir, folder_name, storage_saved, created_at, updated_at
			`
	rows, err := db.NamedQueryContext(ctx, stmt, entity)
	if err != nil {
		return err
	}
	defer rows.Close()

	if !rows.Next() {
		return fmt.Errorf("no rows returned for update of entity with id %d", entity.Id)
	}
	if err := rows.StructScan(entity); err != nil {
		return err
	}
	return nil
}

func (r *repo) UpdateStorageSavedByTwitterId(ctx context.Context, db *sqlx.DB, twitterId uint64, saved bool) error {
	stmt := `UPDATE lst_entities SET storage_saved=?, updated_at=CURRENT_TIMESTAMP WHERE lst_id=?`
	_, err := db.ExecContext(ctx, stmt, saved, twitterId)
	return err
}

////////////////////////////////////////////////////////////////////////////////

func (r *repo) Delete(ctx context.Context, db *sqlx.DB, id int) error {
	stmt := `DELETE FROM lst_entities WHERE id=?`
	_, err := db.ExecContext(ctx, stmt, id)
	return err
}
