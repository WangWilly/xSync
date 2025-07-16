package tokenrepo

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/WangWilly/xSync/pkgs/commonpkg/model"
	"github.com/jmoiron/sqlx"
)

// GetByAddress retrieves a token by its address
func (r *repo) GetByAddress(ctx context.Context, db *sqlx.DB, address string) (*model.Token, error) {
	query := `
		SELECT id, address, chain_id, decimals, name, symbol, logo_uri, tags, daily, extensions, 
			   created_at, updated_at, chroma_embedded, chroma_document_id
		FROM tokens 
		WHERE address = $1
	`

	token := &model.Token{}
	err := db.GetContext(ctx, token, query, address)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("token not found: %s", address)
		}
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	return token, nil
}
