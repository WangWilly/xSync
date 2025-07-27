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
			 RETURNING id, lst_id, name, parent_dir, folder_name, storage_saved, created_at, updated_at
			`
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
			 ON CONFLICT(lst_id, parent_dir) DO UPDATE SET name=:name, folder_name=:folder_name, storage_saved=:storage_saved, updated_at=CURRENT_TIMESTAMP
			 RETURNING id, lst_id, name, parent_dir, folder_name, storage_saved, created_at, updated_at
			`
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
	stmt := `SELECT * FROM lst_entities WHERE id = :id`
	rows, err := db.NamedQueryContext(ctx, stmt, model.ListEntity{
		Id: sql.NullInt32{Int32: int32(id), Valid: true},
	})
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, nil
	}
	result := &model.ListEntity{}
	if err := rows.StructScan(result); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *repo) Get(ctx context.Context, db *sqlx.DB, lid int64, parentDir string) (*model.ListEntity, error) {
	parentDir, err := filepath.Abs(parentDir)
	if err != nil {
		return nil, err
	}

	stmt := `SELECT * FROM lst_entities WHERE lst_id = :lst_id AND parent_dir = :parent_dir`
	rows, err := db.NamedQueryContext(ctx, stmt, model.ListEntity{
		LstId:     lid,
		ParentDir: parentDir,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, nil
	}
	result := &model.ListEntity{}
	if err := rows.StructScan(result); err != nil {
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
			 WHERE id=:id
			 RETURNING id, lst_id, name, parent_dir, folder_name, storage_saved, created_at, updated_at
			`
	rows, err := db.NamedQueryContext(ctx, stmt, entity)
	if err != nil {
		return err
	}
	defer rows.Close()

	if !rows.Next() {
		return fmt.Errorf("no rows returned for update of entity with id %d", entity.Id.Int32)
	}
	if err := rows.StructScan(entity); err != nil {
		return err
	}
	return nil
}

func (r *repo) UpdateStorageSavedByTwitterId(ctx context.Context, db *sqlx.DB, twitterId uint64, saved bool) error {
	stmt := `UPDATE lst_entities 
			 SET
			 	storage_saved = :storage_saved,
				updated_at = CURRENT_TIMESTAMP
			 WHERE lst_id = :lst_id
			`
	_, err := db.NamedExecContext(ctx, stmt, model.ListEntity{
		StorageSaved: saved,
		LstId:        int64(twitterId),
	})
	return err
}

////////////////////////////////////////////////////////////////////////////////

func (r *repo) Delete(ctx context.Context, db *sqlx.DB, id int) error {
	stmt := `DELETE FROM lst_entities WHERE id = :id`
	_, err := db.NamedExecContext(ctx, stmt, model.ListEntity{
		Id: sql.NullInt32{Int32: int32(id), Valid: true},
	})
	return err
}
