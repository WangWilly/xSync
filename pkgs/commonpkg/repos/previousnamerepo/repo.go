package previousnamerepo

import (
	"context"

	"github.com/jmoiron/sqlx"
)

type repo struct{}

func New() *repo {
	return &repo{}
}

////////////////////////////////////////////////////////////////////////////////

func (r *repo) Create(
	ctx context.Context,
	db *sqlx.DB,
	uid uint64,
	name string,
	screenName string,
) error {
	stmt := `INSERT INTO user_previous_names (uid, screen_name, name) VALUES($1, $2, $3)`
	_, err := db.ExecContext(ctx, stmt, uid, screenName, name)
	return err
}
