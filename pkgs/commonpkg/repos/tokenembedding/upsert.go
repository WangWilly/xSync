package tokenembedding

import (
	"context"
	"fmt"

	"github.com/WangWilly/xSync/pkgs/commonpkg/model"
	"github.com/jmoiron/sqlx"
)

func (r *repo) Upsert(ctx context.Context, db *sqlx.DB, tokenAddress, chromaDocumentID, embeddingContent string) error {
	query := `
		INSERT INTO token_embeddings (token_address, chroma_document_id, embedding_content)
		VALUES (:token_address, :chroma_document_id, :embedding_content)
		ON CONFLICT (token_address) DO UPDATE SET
			chroma_document_id=:chroma_document_id,
			embedding_content=:embedding_content,
			updated_at=CURRENT_TIMESTAMP
		RETURNING id, token_address, chroma_document_id, embedding_content, created_at, updated_at
	`
	rows, err := db.NamedQueryContext(ctx, query, &model.TokenEmbedding{
		TokenAddress:     tokenAddress,
		ChromaDocumentID: chromaDocumentID,
		EmbeddingContent: embeddingContent,
	})
	if err != nil {
		return err
	}
	defer rows.Close()

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
		INSERT INTO token_embeddings (token_address, chroma_document_id, embedding_content)
		VALUES (:token_address, :chroma_document_id, :embedding_content)
		ON CONFLICT (token_address) DO UPDATE SET
			chroma_document_id=:chroma_document_id,
			embedding_content=:embedding_content,
			updated_at=CURRENT_TIMESTAMP
		RETURNING id, token_address, chroma_document_id, embedding_content, created_at, updated_at
	`
	stmt, err := tx.PrepareNamedContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for i := range embeddings {
		err := stmt.
			QueryRowxContext(ctx, embeddings[i]).
			StructScan(embeddings[i])
		if err != nil {
			return fmt.Errorf("failed to upsert token embedding: %w", err)
		}
	}

	return tx.Commit()
}
