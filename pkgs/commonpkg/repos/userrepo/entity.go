package userrepo

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"time"

	"github.com/WangWilly/xSync/pkgs/commonpkg/model"
	"github.com/jmoiron/sqlx"
)

func (r *Repo) CreateEntity(db *sqlx.DB, entity *model.UserEntity) error {
	abs, err := filepath.Abs(entity.ParentDir)
	if err != nil {
		return err
	}
	entity.ParentDir = abs

	stmt := `INSERT INTO user_entities(user_id, name, parent_dir, folder_name, storage_saved)
			VALUES(:user_id, :name, :parent_dir, :folder_name, :storage_saved)
			RETURNING id, user_id, name, parent_dir, folder_name, storage_saved, media_count, latest_release_time, created_at, updated_at`
	rows, err := db.NamedQuery(stmt, entity)
	if err != nil {
		return err
	}
	defer rows.Close()

	if !rows.Next() {
		return fmt.Errorf("no rows returned for user entity with user_id %d", entity.Uid)
	}
	return nil
}

func (r *Repo) UpsertEntity(db *sqlx.DB, entity *model.UserEntity) error {
	abs, err := filepath.Abs(entity.ParentDir)
	if err != nil {
		return err
	}
	entity.ParentDir = abs

	stmt := `INSERT INTO user_entities(user_id, name, parent_dir, folder_name, storage_saved)
		VALUES(:user_id, :name, :parent_dir, :folder_name, :storage_saved)
		ON CONFLICT(user_id) DO UPDATE SET name=:name, parent_dir=:parent_dir, folder_name=:folder_name, storage_saved=:storage_saved, updated_at=CURRENT_TIMESTAMP
		RETURNING id, user_id, name, parent_dir, folder_name, storage_saved, media_count, latest_release_time, created_at, updated_at`
	rows, err := db.NamedQuery(stmt, entity)
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

func (r *Repo) GetEntity(db *sqlx.DB, uid uint64, parentDir string) (*model.UserEntity, error) {
	parentDir, err := filepath.Abs(parentDir)
	if err != nil {
		return nil, err
	}

	stmt := `SELECT * FROM user_entities WHERE user_id=? AND parent_dir=?`
	result := &model.UserEntity{}
	err = db.Get(result, stmt, uid, parentDir)
	if err == sql.ErrNoRows {
		err = nil
		result = nil
	}
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (r *Repo) GetEntityById(db *sqlx.DB, id int) (*model.UserEntity, error) {
	stmt := `SELECT * FROM user_entities WHERE id=?`
	result := &model.UserEntity{}
	err := db.Get(result, stmt, id)
	if err == sql.ErrNoRows {
		result = nil
		err = nil
	}

	if err != nil {
		return nil, err
	}
	return result, nil
}

func (r *Repo) GetEntityByTwitterId(db *sqlx.DB, twitterId uint64) (*model.UserEntity, error) {
	stmt := `SELECT * FROM user_entities WHERE user_id=?`
	result := &model.UserEntity{}
	err := db.Get(result, stmt, twitterId)
	if err == sql.ErrNoRows {
		result = nil
		err = nil
	}

	if err != nil {
		return nil, err
	}
	return result, nil
}

////////////////////////////////////////////////////////////////////////////////

func (r *Repo) UpdateEntity(db *sqlx.DB, entity *model.UserEntity) error {
	stmt := `UPDATE user_entities SET name=?, parent_dir=?, folder_name=?, storage_saved=?, media_count=?, latest_release_time=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`
	_, err := db.Exec(stmt, entity.Name, entity.ParentDir, entity.FolderName, entity.StorageSaved, entity.MediaCount, entity.LatestReleaseTime, entity.Id)
	return err
}

func (r *Repo) UpdateEntityMediaCount(db *sqlx.DB, eid int, count int) error {
	stmt := `UPDATE user_entities SET media_count=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`
	_, err := db.Exec(stmt, count, eid)
	return err
}

func (r *Repo) UpdateEntityTweetStat(db *sqlx.DB, eid int, baseline time.Time, count int) error {
	stmt := `UPDATE user_entities SET latest_release_time=?, media_count=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`
	_, err := db.Exec(stmt, baseline, count, eid)
	return err
}

func (r *Repo) UpdateEntityStorageSavedByTwitterId(db *sqlx.DB, twitterId uint64, saved bool) error {
	stmt := `UPDATE user_entities SET storage_saved=?, updated_at=CURRENT_TIMESTAMP WHERE user_id=?`
	_, err := db.Exec(stmt, saved, twitterId)
	return err
}

func (r *Repo) SetEntityLatestReleaseTime(db *sqlx.DB, id int, t time.Time) error {
	stmt := `UPDATE user_entities SET latest_release_time=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`
	_, err := db.Exec(stmt, t, id)
	return err
}

////////////////////////////////////////////////////////////////////////////////

func (r *Repo) DeleteEntity(db *sqlx.DB, id uint32) error {
	stmt := `DELETE FROM user_entities WHERE id=?`
	_, err := db.Exec(stmt, id)
	return err
}
