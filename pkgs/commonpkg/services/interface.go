package services

import (
	"context"

	"github.com/WangWilly/xSync/pkgs/commonpkg/clients/juptokenclient"
	"github.com/WangWilly/xSync/pkgs/commonpkg/model"
	"github.com/jmoiron/sqlx"

	chroma "github.com/amikos-tech/chroma-go/pkg/api/v2"
)

type JupTokenClient interface {
	GetAllTokens(ctx context.Context) ([]juptokenclient.JupTokenDto, error)
	GetTokenStats(ctx context.Context) (*juptokenclient.JupTokenStatsDto, error)
}

type ChromaTokenClient interface {
	GetCollectionCount(ctx context.Context) (int, error)
	// CreateTokenFromJupiter(ctx context.Context, tokenInfo *juptokenclient.JupTokenDto) (string, error)
	BatchCreateTokensFromJupiter(ctx context.Context, tokens []juptokenclient.JupTokenDto) ([]string, error)
	GetTokens(ctx context.Context, query string, limit int) (chroma.QueryResult, error)
}

type TokenRepo interface {
	GetByAddress(ctx context.Context, db *sqlx.DB, address string) (*model.Token, error)
	ListByNotInChroma(ctx context.Context, db *sqlx.DB, limit int) ([]model.Token, error)
	// MarkTokenAsEmbedded(ctx context.Context, db *sqlx.DB, address string, chromaDocumentID string) error
	SearchTokens(ctx context.Context, db *sqlx.DB, searchTerm string, limit int) ([]model.Token, error)
	TotalCount(ctx context.Context, db *sqlx.DB) (int, error)
	BatchUpsertFromJupTokenDto(ctx context.Context, db *sqlx.DB, tokens []juptokenclient.JupTokenDto) ([]model.Token, error)
	BatchMarkTokenAsEmbedded(ctx context.Context, db *sqlx.DB, tokens []string, chromaDocumentIDs []string) error
}

type TokenEmbeddingRepo interface {
	// Upsert(ctx context.Context, db *sqlx.DB, tokenAddress string, chromaDocumentID string, embeddingContent string) error
	BatchUpsert(ctx context.Context, db *sqlx.DB, embeddings []*model.TokenEmbedding) error
}
