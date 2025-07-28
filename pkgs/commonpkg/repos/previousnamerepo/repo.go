package previousnamerepo

import (
	"context"

	"github.com/WangWilly/xSync/pkgs/commonpkg/model"
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
	stmt := `INSERT INTO user_previous_names (uid, screen_name, name)
			 VALUES(:uid, :screen_name, :name)
			`
	_, err := db.NamedExecContext(ctx, stmt, model.UserPreviousName{
		Uid:        uid,
		ScreenName: screenName,
		Name:       name,
	})
	return err
}
