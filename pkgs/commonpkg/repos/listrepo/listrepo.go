package listrepo

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

func (r *Repo) Create(db *sqlx.DB, lst *model.List) error {
	stmt := `INSERT INTO lsts(id, name, owner_uid) 
			VALUES(:id, :name, :owner_uid)
			RETURNING id, name, owner_uid, created_at, updated_at`
	rows, err := db.NamedQuery(stmt, lst)
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

func (r *Repo) Upsert(db *sqlx.DB, lst *model.List) error {
	stmt := `INSERT INTO lsts(id, name, owner_uid)
			VALUES(:id, :name, :owner_uid)
			ON CONFLICT(id) DO UPDATE SET name=:name, updated_at=CURRENT_TIMESTAMP
			RETURNING id, name, owner_uid, created_at, updated_at`
	rows, err := db.NamedQuery(stmt, &lst)
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

func (r *Repo) GetById(db *sqlx.DB, lid uint64) (*model.List, error) {
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

func (r *Repo) Delete(db *sqlx.DB, lid uint64) error {
	stmt := `DELETE FROM lsts WHERE id=$1`
	_, err := db.Exec(stmt, lid)
	return err
}

////////////////////////////////////////////////////////////////////////////////

func (r *Repo) Update(db *sqlx.DB, lst *model.List) error {
	stmt := `UPDATE lsts SET name=$1, updated_at=CURRENT_TIMESTAMP WHERE id=$2`
	_, err := db.Exec(stmt, lst.Name, lst.Id)
	return err
}
