package userrepo

import (
	"database/sql"
	"fmt"

	"github.com/WangWilly/xSync/pkgs/commonpkg/model"
	"github.com/jmoiron/sqlx"
)

func (r *Repo) CreateLink(db *sqlx.DB, lnk *model.UserLink) error {
	stmt := `INSERT INTO user_links(user_id, name, parent_lst_entity_id, storage_saved)
			VALUES(:user_id, :name, :parent_lst_entity_id, :storage_saved)
			RETURNING id, user_id, name, parent_lst_entity_id, storage_saved, created_at, updated_at`
	rows, err := db.NamedQuery(stmt, lnk)
	if err != nil {
		return err
	}
	defer rows.Close()

	if !rows.Next() {
		return fmt.Errorf("no rows returned for user link with user_id %d", lnk.UserTwitterId)
	}
	if err := rows.StructScan(lnk); err != nil {
		return err
	}
	return nil
}

func (r *Repo) UpsertLink(db *sqlx.DB, lnk *model.UserLink) error {
	stmt := `INSERT INTO user_links(user_id, name, parent_lst_entity_id, storage_saved)
			VALUES(:user_id, :name, :parent_lst_entity_id, :storage_saved)
			ON CONFLICT(user_id, parent_lst_entity_id) DO UPDATE SET name = :name, storage_saved = :storage_saved, updated_at=CURRENT_TIMESTAMP
			RETURNING id, user_id, name, parent_lst_entity_id, storage_saved, created_at, updated_at`
	rows, err := db.NamedQuery(stmt, lnk)
	if err != nil {
		return err
	}
	defer rows.Close()

	if !rows.Next() {
		return fmt.Errorf("no rows returned for upsert of user link with user_id %d", lnk.UserTwitterId)
	}
	if err := rows.StructScan(lnk); err != nil {
		return err
	}
	return nil
}

////////////////////////////////////////////////////////////////////////////////

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

////////////////////////////////////////////////////////////////////////////////

func (r *Repo) UpdateLink(db *sqlx.DB, id int32, name string) error {
	stmt := `UPDATE user_links SET name = ?, updated_at=CURRENT_TIMESTAMP WHERE id = ?`
	_, err := db.Exec(stmt, name, id)
	return err
}

func (r *Repo) UpdateLinkStorageSaved(db *sqlx.DB, id int32, storageSaved bool) error {
	stmt := `UPDATE user_links SET storage_saved = ?, updated_at=CURRENT_TIMESTAMP WHERE id = ?`
	_, err := db.Exec(stmt, storageSaved, id)
	if err != nil {
		return err
	}
	return nil
}

////////////////////////////////////////////////////////////////////////////////

func (r *Repo) DeleteLink(db *sqlx.DB, id int32) error {
	stmt := `DELETE FROM user_links WHERE id = ?`
	_, err := db.Exec(stmt, id)
	return err
}
