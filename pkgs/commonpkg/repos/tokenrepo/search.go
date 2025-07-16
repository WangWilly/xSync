package tokenrepo

import (
	"context"
	"fmt"

	"github.com/WangWilly/xSync/pkgs/commonpkg/model"
	"github.com/jmoiron/sqlx"
)

// SearchTokens searches tokens by name or symbol
func (r *repo) SearchTokens(ctx context.Context, db *sqlx.DB, searchTerm string, limit int) ([]model.Token, error) {
	query := `
		SELECT id, address, chain_id, decimals, name, symbol, logo_uri, tags, daily, extensions, 
			   created_at, updated_at, chroma_embedded, chroma_document_id
		FROM tokens 
		WHERE name ILIKE $1 OR symbol ILIKE $1
		ORDER BY 
			CASE 
				WHEN symbol ILIKE $1 THEN 1
				WHEN name ILIKE $1 THEN 2
				ELSE 3
			END,
			name
		LIMIT $2
	`

	var tokens []model.Token
	searchPattern := "%" + searchTerm + "%"
	err := db.SelectContext(ctx, &tokens, query, searchPattern, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search tokens: %w", err)
	}

	return tokens, nil
}
