package userrepo

import (
	"database/sql"
	"fmt"

	"github.com/WangWilly/xSync/pkgs/commonpkg/model"
	"github.com/jmoiron/sqlx"
)

////////////////////////////////////////////////////////////////////////////////

type Repo struct{}

func New() *Repo {
	return &Repo{}
}

////////////////////////////////////////////////////////////////////////////////

func (r *Repo) Create(db *sqlx.DB, usr *model.User) error {
	stmt := `INSERT INTO users(id, screen_name, name, protected, friends_count) 
			VALUES(:id, :screen_name, :name, :protected, :friends_count)
			RETURNING id, screen_name, name, protected, friends_count, created_at, updated_at`
	rows, err := db.NamedQuery(stmt, usr)
	if err != nil {
		return err
	}
	defer rows.Close()

	if !rows.Next() {
		return fmt.Errorf("no rows returned for user with id %d", usr.Id)
	}
	if err := rows.StructScan(usr); err != nil {
		return err
	}
	return nil
}

func (r *Repo) Upsert(db *sqlx.DB, usr *model.User) error {
	stmt := `INSERT INTO users(id, screen_name, name, protected, friends_count) VALUES(:id, :screen_name, :name, :protected, :friends_count)
			ON CONFLICT(id) DO UPDATE SET screen_name=:screen_name, name=:name, protected=:protected, friends_count=:friends_count, updated_at=CURRENT_TIMESTAMP
			RETURNING id, screen_name, name, protected, friends_count, created_at, updated_at`
	rows, err := db.NamedQuery(stmt, usr)
	if err != nil {
		return err
	}
	defer rows.Close()

	if !rows.Next() {
		return fmt.Errorf("no rows returned for upsert of user with id %d", usr.Id)
	}
	if err := rows.StructScan(usr); err != nil {
		return err
	}
	return nil
}

////////////////////////////////////////////////////////////////////////////////

func (r *Repo) GetById(db *sqlx.DB, uid uint64) (*model.User, error) {
	stmt := `SELECT * FROM users WHERE id=$1`
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

////////////////////////////////////////////////////////////////////////////////

func (r *Repo) Update(db *sqlx.DB, usr *model.User) error {
	stmt := `UPDATE users SET screen_name=:screen_name, name=:name, protected=:protected, friends_count=:friends_count, updated_at=CURRENT_TIMESTAMP WHERE id=:id`
	_, err := db.NamedExec(stmt, usr)
	return err
}

////////////////////////////////////////////////////////////////////////////////

func (r *Repo) Delete(db *sqlx.DB, uid uint64) error {
	stmt := `DELETE FROM users WHERE id=$1`
	_, err := db.Exec(stmt, uid)
	return err
}

////////////////////////////////////////////////////////////////////////////////

// TODO:
func (r *Repo) CreatePreviousName(db *sqlx.DB, uid uint64, name string, screenName string) error {
	stmt := `INSERT INTO user_previous_names(uid, screen_name, name) VALUES($1, $2, $3)`
	_, err := db.Exec(stmt, uid, screenName, name)
	return err
}
