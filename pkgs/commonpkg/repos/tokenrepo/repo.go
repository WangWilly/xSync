package tokenrepo

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
)

type repo struct {
}

func New() *repo {
	return &repo{}
}

////////////////////////////////////////////////////////////////////////////////

func (r *repo) TotalCount(ctx context.Context, db *sqlx.DB) (int, error) {
	var count int
	err := db.GetContext(ctx, &count, "SELECT COUNT(*) FROM tokens")
	if err != nil {
		return 0, fmt.Errorf("failed to get token count: %w", err)
	}

	return count, nil
}
