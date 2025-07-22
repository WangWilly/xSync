package userrepo

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/WangWilly/xSync/pkgs/commonpkg/model"
	"github.com/jmoiron/sqlx"
)

////////////////////////////////////////////////////////////////////////////////

type repo struct{}

func New() *repo {
	return &repo{}
}

////////////////////////////////////////////////////////////////////////////////

func (r *repo) Create(ctx context.Context, db *sqlx.DB, usr *model.User) error {
	stmt := `INSERT INTO users(id, screen_name, name, protected, friends_count) 
			VALUES(:id, :screen_name, :name, :protected, :friends_count)
			RETURNING id, screen_name, name, protected, friends_count, created_at, updated_at`
	rows, err := db.NamedQueryContext(ctx, stmt, usr)
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

func (r *repo) Upsert(ctx context.Context, db *sqlx.DB, usr *model.User) error {
	stmt := `INSERT INTO users(id, screen_name, name, protected, friends_count) VALUES(:id, :screen_name, :name, :protected, :friends_count)
			ON CONFLICT(id) DO UPDATE SET screen_name=:screen_name, name=:name, protected=:protected, friends_count=:friends_count, updated_at=CURRENT_TIMESTAMP
			RETURNING id, screen_name, name, protected, friends_count, created_at, updated_at`
	rows, err := db.NamedQueryContext(ctx, stmt, usr)
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

func (r *repo) GetById(ctx context.Context, db *sqlx.DB, uid uint64) (*model.User, error) {
	stmt := `SELECT * FROM users WHERE id=$1`
	result := &model.User{}
	err := db.GetContext(ctx, result, stmt, uid)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return result, nil
}

////////////////////////////////////////////////////////////////////////////////

func (r *repo) Update(ctx context.Context, db *sqlx.DB, usr *model.User) error {
	stmt := `UPDATE users SET screen_name=:screen_name, name=:name, protected=:protected, friends_count=:friends_count, updated_at=CURRENT_TIMESTAMP WHERE id=:id`
	_, err := db.NamedExecContext(ctx, stmt, usr)
	return err
}

////////////////////////////////////////////////////////////////////////////////

func (r *repo) Delete(ctx context.Context, db *sqlx.DB, uid uint64) error {
	stmt := `DELETE FROM users WHERE id=$1`
	_, err := db.ExecContext(ctx, stmt, uid)
	return err
}
