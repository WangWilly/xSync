package listrepo

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

func (r *repo) Create(ctx context.Context, db *sqlx.DB, lst *model.List) error {
	stmt := `INSERT INTO lsts(id, name, owner_uid) 
			VALUES(:id, :name, :owner_uid)
			RETURNING id, name, owner_uid, created_at, updated_at`
	rows, err := db.NamedQueryContext(ctx, stmt, lst)
	if err != nil {
		return err
	}
	defer rows.Close()

	if !rows.Next() {
		return fmt.Errorf("no rows returned for list with id %d", lst.Id)
	}
	if err := rows.StructScan(lst); err != nil {
		return err
	}
	return nil
}

func (r *repo) Upsert(ctx context.Context, db *sqlx.DB, lst *model.List) error {
	stmt := `INSERT INTO lsts(id, name, owner_uid)
			VALUES(:id, :name, :owner_uid)
			ON CONFLICT(id) DO UPDATE SET name=:name, updated_at=CURRENT_TIMESTAMP
			RETURNING id, name, owner_uid, created_at, updated_at`
	rows, err := db.NamedQueryContext(ctx, stmt, &lst)
	if err != nil {
		return err
	}
	defer rows.Close()

	if !rows.Next() {
		return fmt.Errorf("no rows returned for upsert of list with id %d", lst.Id)
	}
	if err := rows.StructScan(lst); err != nil {
		return err
	}
	return nil
}

////////////////////////////////////////////////////////////////////////////////

func (r *repo) GetById(ctx context.Context, db *sqlx.DB, lid uint64) (*model.List, error) {
	stmt := `SELECT * FROM lsts WHERE id = $1`
	result := &model.List{}
	err := db.Get(result, stmt, lid)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return result, nil
}

////////////////////////////////////////////////////////////////////////////////

func (r *repo) Delete(ctx context.Context, db *sqlx.DB, lid uint64) error {
	stmt := `DELETE FROM lsts WHERE id=$1`
	_, err := db.ExecContext(ctx, stmt, lid)
	return err
}

////////////////////////////////////////////////////////////////////////////////

func (r *repo) Update(ctx context.Context, db *sqlx.DB, lst *model.List) error {
	stmt := `UPDATE lsts SET name=$1, updated_at=CURRENT_TIMESTAMP WHERE id=$2`
	_, err := db.ExecContext(ctx, stmt, lst.Name, lst.Id)
	return err
}
