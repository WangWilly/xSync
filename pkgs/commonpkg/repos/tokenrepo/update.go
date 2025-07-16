package tokenrepo

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

func (r *repo) MarkTokenAsEmbedded(ctx context.Context, db *sqlx.DB, address, chromaDocumentID string) error {
	query := `
		UPDATE tokens 
		SET chroma_embedded = TRUE, chroma_document_id = $1, updated_at = $2
		WHERE address = $3
	`

	_, err := db.ExecContext(ctx, query, chromaDocumentID, time.Now(), address)
	if err != nil {
		return fmt.Errorf("failed to mark token as embedded: %w", err)
	}

	return nil
}

func (r *repo) BatchMarkTokenAsEmbedded(ctx context.Context, db *sqlx.DB, tokens []string, chromaDocumentIDs []string) error {
	if len(tokens) == 0 || len(chromaDocumentIDs) == 0 {
		return nil
	}
	if len(tokens) != len(chromaDocumentIDs) {
		return fmt.Errorf("tokens and chromaDocumentIDs must have the same length")
	}

	// Create a transaction for batching all inserts
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	query := `
		UPDATE tokens 
		SET chroma_embedded = TRUE, chroma_document_id = $1, updated_at = $2
		WHERE address = $3
	`

	stmt, err := tx.PreparexContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	now := time.Now()
	for i, address := range tokens {
		_, err := stmt.ExecContext(ctx, chromaDocumentIDs[i], now, address)
		if err != nil {
			return fmt.Errorf("failed to mark token %s as embedded: %w", address, err)
		}
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
