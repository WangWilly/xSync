package services

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/WangWilly/xSync/pkgs/commonpkg/clients/juptokenclient"
	"github.com/WangWilly/xSync/pkgs/commonpkg/model"
	"github.com/jmoiron/sqlx"
	log "github.com/sirupsen/logrus"
)

// TokenService orchestrates token collection from Jupiter API to PostgreSQL to ChromaDB
type TokenService struct {
	db *sqlx.DB

	jupiterClient      JupTokenClient
	chromaClient       ChromaTokenClient
	tokenRepo          TokenRepo
	tokenEmbeddingRepo TokenEmbeddingRepo
	logger             *log.Entry
}

// NewTokenService creates a new token service
func NewTokenService(
	db *sqlx.DB,
	jupiterClient JupTokenClient,
	chromaClient ChromaTokenClient,
	tokenRepo TokenRepo,
	tokenEmbeddingRepo TokenEmbeddingRepo,
) *TokenService {
	return &TokenService{
		db:                 db,
		jupiterClient:      jupiterClient,
		chromaClient:       chromaClient,
		tokenRepo:          tokenRepo,
		tokenEmbeddingRepo: tokenEmbeddingRepo,
		logger:             log.WithField("service", "token_service"),
	}
}

// CollectAndStoreTokens collects tokens from Jupiter API and stores them in PostgreSQL and ChromaDB
func (s *TokenService) CollectAndStoreTokens(ctx context.Context) error {
	s.logger.Info("Starting token collection process")

	s.logger.Info("Fetching tokens from Jupiter API...")
	tokens, err := s.jupiterClient.GetAllTokens(ctx)
	if err != nil {
		return fmt.Errorf("failed to get tokens from Jupiter API: %w", err)
	}

	s.logger.Infof("Retrieved %d tokens from Jupiter API", len(tokens))

	s.logger.Info("Saving tokens to PostgreSQL...")
	savedCount := 0
	for tokenBatch := range slices.Chunk(tokens, 100) {
		// Save token directly as it's already in the correct format
		_, err := s.tokenRepo.BatchUpsertFromJupTokenDto(ctx, s.db, tokenBatch)
		if err != nil {
			s.logger.WithError(err).Warnf("Failed to upsert token batch: %v", err)
			continue
		}
		savedCount++

		// Log progress every 100 tokens
		if savedCount == 0 {
			s.logger.Infof("Saved %d tokens to PostgreSQL", savedCount)
		}
	}

	s.logger.Infof("Saved %d tokens to PostgreSQL", savedCount)

	s.logger.Info("Processing tokens for ChromaDB...")
	return s.processTokensForChroma(ctx)
}

// processTokensForChroma processes tokens that haven't been embedded in ChromaDB yet
func (s *TokenService) processTokensForChroma(ctx context.Context) error {
	batchSize := 50 // Process in batches to avoid overwhelming the system

	for {
		// Get tokens not yet in ChromaDB
		tokens, err := s.tokenRepo.ListByNotInChroma(ctx, s.db, batchSize)
		if err != nil {
			return fmt.Errorf("failed to get tokens not in chroma: %w", err)
		}

		if len(tokens) == 0 {
			s.logger.Info("No more tokens to process for ChromaDB")
			break
		}

		s.logger.Infof("Processing %d tokens for ChromaDB", len(tokens))

		// // Process each token
		// for _, token := range tokens {
		// 	// Convert model.Token to juptokenclient.JupTokenDto for ChromaDB client
		// 	tokenInfo := &juptokenclient.JupTokenDto{
		// 		Address:  token.Address,
		// 		Decimals: token.Decimals,
		// 		Name:     token.Name,
		// 		Symbol:   token.Symbol,
		// 		LogoURI:  token.LogoURI,
		// 		Tags:     parseTagsFromJSON(token.Tags),
		// 	}

		// 	// Add to ChromaDB
		// 	chromaDocID, err := s.chromaClient.CreateTokenFromJupiter(ctx, tokenInfo)
		// 	if err != nil {
		// 		s.logger.WithError(err).Warnf("Failed to add token %s to ChromaDB", token.Address)
		// 		continue
		// 	}

		// 	// Mark as embedded in PostgreSQL
		// 	err = s.tokenRepo.MarkTokenAsEmbedded(ctx, s.db, token.Address, chromaDocID)
		// 	if err != nil {
		// 		s.logger.WithError(err).Warnf("Failed to mark token %s as embedded", token.Address)
		// 		continue
		// 	}

		// 	// Save embedding record

		// 	err = s.tokenEmbeddingRepo.Upsert(ctx, s.db, token.Address, chromaDocID, embeddingContent)
		// 	if err != nil {
		// 		s.logger.WithError(err).Warnf("Failed to save embedding record for token %s", token.Address)
		// 	}
		// }

		// Add small delay between batches
		defer time.Sleep(100 * time.Millisecond)

		tokenInfos := juptokenclient.BatchNewJupTokenDtoFromModel(tokens)
		chromaDocIDs, err := s.chromaClient.BatchCreateTokensFromJupiter(ctx, tokenInfos)
		if err != nil {
			s.logger.WithError(err).Warn("Failed to add token batch to ChromaDB")
			continue
		}
		addrs := make([]string, len(tokens))
		for i, token := range tokens {
			addrs[i] = token.Address
		}
		if err := s.tokenRepo.BatchMarkTokenAsEmbedded(ctx, s.db, addrs, chromaDocIDs); err != nil {
			s.logger.WithError(err).Warn("Failed to mark token batch as embedded")
			continue
		}
		embeddings, err := model.BatchNewTokenEmbeddingsFromTokens(tokens, chromaDocIDs)
		if err != nil {
			s.logger.WithError(err).Warn("Failed to create token embeddings")
			continue
		}
		if err := s.tokenEmbeddingRepo.BatchUpsert(ctx, s.db, embeddings); err != nil {
			s.logger.WithError(err).Warn("Failed to upsert token embeddings")
			continue
		}
	}

	return nil
}

// GetTokenStats returns statistics about the token collection
func (s *TokenService) GetTokenStats(ctx context.Context) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Get PostgreSQL stats
	totalTokens, err := s.tokenRepo.TotalCount(ctx, s.db)
	if err != nil {
		return nil, fmt.Errorf("failed to get token count: %w", err)
	}
	stats["postgres_total_tokens"] = totalTokens

	// Get ChromaDB stats
	chromaCount, err := s.chromaClient.GetCollectionCount(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get chroma collection count: %w", err)
	}
	stats["chroma_total_tokens"] = chromaCount

	// Get Jupiter API stats
	jupiterStats, err := s.jupiterClient.GetTokenStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get jupiter stats: %w", err)
	}
	stats["jupiter_stats"] = jupiterStats

	return stats, nil
}

// SearchTokens searches for tokens using both PostgreSQL and ChromaDB
func (s *TokenService) SearchTokens(ctx context.Context, query string, useChroma bool) (interface{}, error) {
	if useChroma {
		results, err := s.chromaClient.GetTokens(ctx, query, 10)
		if err != nil {
			return nil, fmt.Errorf("failed to search tokens in chroma: %w", err)
		}
		return results, nil
	} else {
		// Use PostgreSQL for exact search
		results, err := s.tokenRepo.SearchTokens(ctx, s.db, query, 10)
		if err != nil {
			return nil, fmt.Errorf("failed to search tokens in postgres: %w", err)
		}
		return results, nil
	}
}
