package userrepo

import (
	"database/sql"
	"path/filepath"
	"time"

	"github.com/WangWilly/xSync/pkgs/commonpkg/model"
	"github.com/jmoiron/sqlx"
)

type Repo struct{}

func New() *Repo {
	return &Repo{}
}

func (r *Repo) Create(db *sqlx.DB, usr *model.User) error {
	stmt := `INSERT INTO users(id, screen_name, name, protected, friends_count) VALUES(:id, :screen_name, :name, :protected, :friends_count)`
	_, err := db.NamedExec(stmt, usr)
	return err
}

func (r *Repo) Delete(db *sqlx.DB, uid uint64) error {
	stmt := `DELETE FROM users WHERE id=?`
	_, err := db.Exec(stmt, uid)
	return err
}

func (r *Repo) GetById(db *sqlx.DB, uid uint64) (*model.User, error) {
	stmt := `SELECT * FROM users WHERE id=?`
	result := &model.User{}
	err := db.Get(result, stmt, uid)
	if err == sql.ErrNoRows {
		result = nil
		err = nil
	}
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (r *Repo) Update(db *sqlx.DB, usr *model.User) error {
	stmt := `UPDATE users SET screen_name=:screen_name, name=:name, protected=:protected, friends_count=:friends_count, updated_at=CURRENT_TIMESTAMP WHERE id=:id`
	_, err := db.NamedExec(stmt, usr)
	return err
}

func (r *Repo) CreateEntity(db *sqlx.DB, entity *model.UserEntity) error {
	abs, err := filepath.Abs(entity.ParentDir)
	if err != nil {
		return err
	}
	entity.ParentDir = abs

	stmt := `INSERT INTO user_entities(user_id, name, parent_dir) VALUES(:user_id, :name, :parent_dir)`
	de, err := db.NamedExec(stmt, entity)
	if err != nil {
		return err
	}
	lastId, err := de.LastInsertId()
	if err != nil {
		return err
	}

	entity.Id.Scan(lastId)
	return nil
}

func (r *Repo) DeleteEntity(db *sqlx.DB, id uint32) error {
	stmt := `DELETE FROM user_entities WHERE id=?`
	_, err := db.Exec(stmt, id)
	return err
}

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
	result := &model.UserEntity{}
	stmt := `SELECT * FROM user_entities WHERE id=?`
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

func (r *Repo) UpdateEntity(db *sqlx.DB, entity *model.UserEntity) error {
	stmt := `UPDATE user_entities SET name=?, latest_release_time=?, media_count=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`
	_, err := db.Exec(stmt, entity.Name, entity.LatestReleaseTime, entity.MediaCount, entity.Id)
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

func (r *Repo) SetEntityLatestReleaseTime(db *sqlx.DB, id int, t time.Time) error {
	stmt := `UPDATE user_entities SET latest_release_time=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`
	_, err := db.Exec(stmt, t, id)
	return err
}

func (r *Repo) RecordPreviousName(db *sqlx.DB, uid uint64, name string, screenName string) error {
	stmt := `INSERT INTO user_previous_names(uid, screen_name, name) VALUES(?, ?, ?)`
	_, err := db.Exec(stmt, uid, screenName, name)
	return err
}

func (r *Repo) CreateLink(db *sqlx.DB, lnk *model.UserLink) error {
	stmt := `INSERT INTO user_links(user_id, name, parent_lst_entity_id) VALUES(:user_id, :name, :parent_lst_entity_id)`
	res, err := db.NamedExec(stmt, lnk)
	if err != nil {
		return err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return err
	}

	lnk.Id.Scan(id)
	return nil
}

func (r *Repo) DeleteLink(db *sqlx.DB, id int32) error {
	stmt := `DELETE FROM user_links WHERE id = ?`
	_, err := db.Exec(stmt, id)
	return err
}

func (r *Repo) GetLinks(db *sqlx.DB, uid uint64) ([]*model.UserLink, error) {
	stmt := `SELECT * FROM user_links WHERE user_id = ?`
	res := []*model.UserLink{}
	err := db.Select(&res, stmt, uid)
	return res, err
}

func (r *Repo) GetLink(db *sqlx.DB, uid uint64, parentLstEntityId int32) (*model.UserLink, error) {
	stmt := `SELECT * FROM user_links WHERE user_id = ? AND parent_lst_entity_id = ?`
	res := &model.UserLink{}
	err := db.Get(res, stmt, uid, parentLstEntityId)
	if err == sql.ErrNoRows {
		err = nil
		res = nil
	}
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (r *Repo) UpdateLink(db *sqlx.DB, id int32, name string) error {
	stmt := `UPDATE user_links SET name = ?, updated_at=CURRENT_TIMESTAMP WHERE id = ?`
	_, err := db.Exec(stmt, name, id)
	return err
}
