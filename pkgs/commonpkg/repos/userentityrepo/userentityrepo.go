package userentityrepo

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"time"

	"github.com/WangWilly/xSync/pkgs/commonpkg/model"
	"github.com/jmoiron/sqlx"
)

type repo struct{}

func New() *repo {
	return &repo{}
}

////////////////////////////////////////////////////////////////////////////////

func (r *repo) Create(ctx context.Context, db *sqlx.DB, entity *model.UserEntity) error {
	abs, err := filepath.Abs(entity.ParentDir)
	if err != nil {
		return err
	}
	entity.ParentDir = abs

	stmt := `INSERT INTO user_entities (user_id, name, parent_dir, folder_name, storage_saved)
			 VALUES (:user_id, :name, :parent_dir, :folder_name, :storage_saved)
			 RETURNING id, user_id, name, parent_dir, folder_name, storage_saved, media_count, latest_release_time, created_at, updated_at
			`
	rows, err := db.NamedQueryContext(ctx, stmt, entity)
	if err != nil {
		return err
	}
	defer rows.Close()

	if !rows.Next() {
		return fmt.Errorf("no rows returned for user entity with user_id %d", entity.Uid)
	}
	if err := rows.StructScan(entity); err != nil {
		return err
	}

	return nil
}

func (r *repo) Upsert(ctx context.Context, db *sqlx.DB, entity *model.UserEntity) error {
	abs, err := filepath.Abs(entity.ParentDir)
	if err != nil {
		return err
	}
	entity.ParentDir = abs

	stmt := `INSERT INTO user_entities (user_id, name, parent_dir, folder_name, storage_saved)
			 VALUES (:user_id, :name, :parent_dir, :folder_name, :storage_saved)
			 ON CONFLICT(user_id) DO UPDATE SET 
			 	name=:name,
				parent_dir=:parent_dir,
				folder_name=:folder_name,
				storage_saved=:storage_saved,
				updated_at=CURRENT_TIMESTAMP
			 RETURNING id, user_id, name, parent_dir, folder_name, storage_saved, media_count, latest_release_time, created_at, updated_at
			`
	rows, err := db.NamedQueryContext(ctx, stmt, entity)
	if err != nil {
		return err
	}
	defer rows.Close()

	if !rows.Next() {
		return fmt.Errorf("no rows returned for upsert of user entity with user_id %d", entity.Uid)
	}
	if err := rows.StructScan(entity); err != nil {
		return err
	}
	return nil
}

////////////////////////////////////////////////////////////////////////////////

func (r *repo) Get(ctx context.Context, db *sqlx.DB, uid uint64, parentDir string) (*model.UserEntity, error) {
	parentDir, err := filepath.Abs(parentDir)
	if err != nil {
		return nil, err
	}

	result := &model.UserEntity{}

	stmt := `SELECT * FROM user_entities WHERE user_id=? AND parent_dir=?`
	err = db.GetContext(ctx, result, stmt, uid, parentDir)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (r *repo) GetById(ctx context.Context, db *sqlx.DB, id int) (*model.UserEntity, error) {
	result := &model.UserEntity{}

	stmt := `SELECT * FROM user_entities WHERE id=?`
	err := db.GetContext(ctx, result, stmt, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (r *repo) GetByTwitterId(ctx context.Context, db *sqlx.DB, twitterId uint64) (*model.UserEntity, error) {
	stmt := `SELECT * FROM user_entities WHERE user_id=?`
	result := &model.UserEntity{}
	err := db.GetContext(ctx, result, stmt, twitterId)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return result, nil
}

////////////////////////////////////////////////////////////////////////////////

func (r *repo) Update(ctx context.Context, db *sqlx.DB, entity *model.UserEntity) error {
	stmt := `UPDATE user_entities 
			SET 
				name=:name,
				parent_dir=:parent_dir,
				folder_name=:folder_name,
				storage_saved=:storage_saved,
				media_count=:media_count,
				latest_release_time=:latest_release_time,
				updated_at=CURRENT_TIMESTAMP
			WHERE id=:id
			RETURNING id, user_id, name, parent_dir, folder_name, storage_saved, media_count, latest_release_time, created_at, updated_at
			`
	rows, err := db.NamedQueryContext(ctx, stmt, entity)
	if err != nil {
		return err
	}
	defer rows.Close()

	if !rows.Next() {
		return fmt.Errorf("no rows returned for update of user entity with id %d", entity.Id)
	}
	if err := rows.StructScan(entity); err != nil {
		return err
	}

	return nil
}

func (r *repo) UpdateTweetStat(ctx context.Context, db *sqlx.DB, eid int, latest_release_time time.Time, count int) error {
	stmt := `UPDATE user_entities
			 SET latest_release_time=?, media_count=?, updated_at=CURRENT_TIMESTAMP
			 WHERE id=?
			`
	_, err := db.ExecContext(ctx, stmt, latest_release_time, count, eid)
	return err
}

func (r *repo) UpdateMediaCount(ctx context.Context, db *sqlx.DB, eid int, count int) error {
	stmt := `UPDATE user_entities
			 SET media_count=?, updated_at=CURRENT_TIMESTAMP
			 WHERE id=?
			`
	_, err := db.ExecContext(ctx, stmt, count, eid)
	return err
}

func (r *repo) UpdateStorageSavedByTwitterId(ctx context.Context, db *sqlx.DB, twitterId uint64, saved bool) error {
	stmt := `UPDATE user_entities
			 SET storage_saved=?, updated_at=CURRENT_TIMESTAMP
			 WHERE user_id=?
			`
	_, err := db.ExecContext(ctx, stmt, saved, twitterId)
	return err
}

////////////////////////////////////////////////////////////////////////////////

func (r *repo) Delete(ctx context.Context, db *sqlx.DB, id uint32) error {
	stmt := `DELETE FROM user_entities WHERE id=?`
	_, err := db.ExecContext(ctx, stmt, id)
	return err
}
