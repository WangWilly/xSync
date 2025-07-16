package model

import (
	"fmt"
	"time"
)

// Token represents a Jupiter token in PostgreSQL
type Token struct {
	ID               int       `db:"id"`
	Address          string    `db:"address"`
	ChainID          int       `db:"chain_id"`
	Decimals         int       `db:"decimals"`
	Name             string    `db:"name"`
	Symbol           string    `db:"symbol"`
	LogoURI          string    `db:"logo_uri"`
	Tags             string    `db:"tags"`       // JSON string
	Daily            string    `db:"daily"`      // JSON string
	Extensions       string    `db:"extensions"` // JSON string
	CreatedAt        time.Time `db:"created_at"`
	UpdatedAt        time.Time `db:"updated_at"`
	ChromaEmbedded   bool      `db:"chroma_embedded"`
	ChromaDocumentID string    `db:"chroma_document_id"`
}

// TokenEmbedding represents token embedding tracking in PostgreSQL
type TokenEmbedding struct {
	ID               int       `db:"id"`
	TokenAddress     string    `db:"token_address"`
	ChromaDocumentID string    `db:"chroma_document_id"`
	EmbeddingContent string    `db:"embedding_content"`
	CreatedAt        time.Time `db:"created_at"`
	UpdatedAt        time.Time `db:"updated_at"`
}

func BatchNewTokenEmbeddingsFromTokens(tokens []Token, chromaDocumentIDs []string) ([]*TokenEmbedding, error) {
	if len(tokens) == 0 || len(chromaDocumentIDs) == 0 {
		return nil, nil
	}
	if len(tokens) != len(chromaDocumentIDs) {
		return nil, fmt.Errorf("tokens and chromaDocumentIDs must have the same length")
	}

	embeddings := make([]*TokenEmbedding, 0, len(tokens))
	for i, token := range tokens {
		embedding := NewTokenEmbeddingFromToken(&token, chromaDocumentIDs[i])
		embeddings = append(embeddings, embedding)
	}

	return embeddings, nil
}

func NewTokenEmbeddingFromToken(token *Token, chromaDocumentID string) *TokenEmbedding {
	embeddingContent := fmt.Sprintf(
		"Token: %s | Symbol: %s | Address: %s",
		token.Name, token.Symbol, token.Address,
	)
	return &TokenEmbedding{
		TokenAddress:     token.Address,
		ChromaDocumentID: chromaDocumentID,
		EmbeddingContent: embeddingContent,
	}
}
