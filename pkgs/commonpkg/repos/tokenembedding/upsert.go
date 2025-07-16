package tokenembedding

import (
	"context"
	"fmt"
	"time"

	"github.com/WangWilly/xSync/pkgs/commonpkg/model"
	"github.com/jmoiron/sqlx"
)

func (r *repo) Upsert(ctx context.Context, db *sqlx.DB, tokenAddress, chromaDocumentID, embeddingContent string) error {
	query := `
		INSERT INTO token_embeddings (token_address, chroma_document_id, embedding_content, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (chroma_document_id) DO UPDATE SET
			embedding_content = EXCLUDED.embedding_content,
			updated_at = EXCLUDED.updated_at
	`

	now := time.Now()
	_, err := db.ExecContext(ctx, query, tokenAddress, chromaDocumentID, embeddingContent, now, now)
	if err != nil {
		return fmt.Errorf("failed to save token embedding: %w", err)
	}

	return nil
}

func (r *repo) BatchUpsert(ctx context.Context, db *sqlx.DB, embeddings []*model.TokenEmbedding) error {
	if len(embeddings) == 0 {
		return nil
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
		INSERT INTO token_embeddings (token_address, chroma_document_id, embedding_content, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (chroma_document_id) DO UPDATE SET
			embedding_content = EXCLUDED.embedding_content,
			updated_at = EXCLUDED.updated_at
		RETURNING token_address, chroma_document_id, embedding_content, created_at, updated_at
	`

	stmt, err := tx.PreparexContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	now := time.Now()
	for _, embedding := range embeddings {
		err := stmt.
			QueryRowxContext(ctx, embedding.TokenAddress, embedding.ChromaDocumentID, embedding.EmbeddingContent, now, now).
			StructScan(embedding)
		if err != nil {
			return fmt.Errorf("failed to upsert token embedding: %w", err)
		}
	}

	return tx.Commit()
}
