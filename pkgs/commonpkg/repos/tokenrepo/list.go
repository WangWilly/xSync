package tokenrepo

import (
	"context"
	"fmt"

	"github.com/WangWilly/xSync/pkgs/commonpkg/model"
	"github.com/jmoiron/sqlx"
)

func (r *repo) List(ctx context.Context, db *sqlx.DB, offset, limit int) ([]model.Token, error) {
	query := `
		SELECT id, address, chain_id, decimals, name, symbol, logo_uri, tags, daily, extensions, 
			   created_at, updated_at, chroma_embedded, chroma_document_id
		FROM tokens 
		ORDER BY created_at DESC
		OFFSET $1 LIMIT $2
	`

	var tokens []model.Token
	err := db.SelectContext(ctx, &tokens, query, offset, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get all tokens: %w", err)
	}

	return tokens, nil
}

// ListByNotInChroma retrieves tokens that haven't been embedded in Chroma
func (r *repo) ListByNotInChroma(ctx context.Context, db *sqlx.DB, limit int) ([]model.Token, error) {
	query := `
		SELECT id, address, chain_id, decimals, name, symbol, logo_uri, tags, daily, extensions, 
			   created_at, updated_at, chroma_embedded, chroma_document_id
		FROM tokens 
		WHERE chroma_embedded = FALSE
		ORDER BY created_at ASC
		LIMIT $1
	`

	var tokens []model.Token
	err := db.SelectContext(ctx, &tokens, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get tokens not in chroma: %w", err)
	}

	return tokens, nil
}
