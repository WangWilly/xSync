package tokenrepo

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/WangWilly/xSync/pkgs/commonpkg/clients/juptokenclient"
	"github.com/WangWilly/xSync/pkgs/commonpkg/model"
	"github.com/jmoiron/sqlx"
)

func (r *repo) UpsertFromJupTokenDto(ctx context.Context, db *sqlx.DB, tokenInfo *juptokenclient.JupTokenDto) (*model.Token, error) {
	// Convert slices/structs to JSON strings
	tagsJSON, err := json.Marshal(tokenInfo.Tags)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal tags: %w", err)
	}

	var dailyJSON, extensionsJSON string
	if tokenInfo.DailyVolume != nil {
		dailyBytes, err := json.Marshal(map[string]interface{}{
			"volume": *tokenInfo.DailyVolume,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to marshal daily: %w", err)
		}
		dailyJSON = string(dailyBytes)
	}

	if tokenInfo.Extensions != nil {
		extensionsBytes, err := json.Marshal(tokenInfo.Extensions)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal extensions: %w", err)
		}
		extensionsJSON = string(extensionsBytes)
	}

	// Try to insert, update if exists
	query := `
		INSERT INTO tokens (address, chain_id, decimals, name, symbol, logo_uri, tags, daily, extensions, created_at, updated_at, chroma_document_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, '')
		ON CONFLICT (address) DO UPDATE SET
			chain_id = EXCLUDED.chain_id,
			decimals = EXCLUDED.decimals,
			name = EXCLUDED.name,
			symbol = EXCLUDED.symbol,
			logo_uri = EXCLUDED.logo_uri,
			tags = EXCLUDED.tags,
			daily = EXCLUDED.daily,
			extensions = EXCLUDED.extensions,
			updated_at = EXCLUDED.updated_at
		RETURNING id, address, chain_id, decimals, name, symbol, logo_uri, tags, daily, extensions, created_at, updated_at, chroma_embedded, chroma_document_id
	`

	now := time.Now()
	token := &model.Token{}

	err = db.QueryRowxContext(ctx, query,
		tokenInfo.Address,
		1, // Default chain ID for Solana
		tokenInfo.Decimals,
		tokenInfo.Name,
		tokenInfo.Symbol,
		tokenInfo.LogoURI,
		string(tagsJSON),
		dailyJSON,
		extensionsJSON,
		now,
		now,
	).StructScan(token)

	if err != nil {
		return nil, fmt.Errorf("failed to save token: %w", err)
	}

	return token, nil
}

func (r *repo) BatchUpsertFromJupTokenDto(ctx context.Context, db *sqlx.DB, tokens []juptokenclient.JupTokenDto) ([]model.Token, error) {
	if len(tokens) == 0 {
		return []model.Token{}, nil
	}

	// Create a transaction for batching all inserts
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	// Prepare the statement once for all inserts
	query := `
		INSERT INTO tokens (address, chain_id, decimals, name, symbol, logo_uri, tags, daily, extensions, created_at, updated_at, chroma_document_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, '')
		ON CONFLICT (address) DO UPDATE SET
			chain_id = EXCLUDED.chain_id,
			decimals = EXCLUDED.decimals,
			name = EXCLUDED.name,
			symbol = EXCLUDED.symbol,
			logo_uri = EXCLUDED.logo_uri,
			tags = EXCLUDED.tags,
			daily = EXCLUDED.daily,
			extensions = EXCLUDED.extensions,
			updated_at = EXCLUDED.updated_at
		RETURNING id, address, chain_id, decimals, name, symbol, logo_uri, tags, daily, extensions, created_at, updated_at, chroma_embedded, chroma_document_id
	`

	stmt, err := tx.PreparexContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	now := time.Now()
	result := make([]model.Token, 0, len(tokens))

	// Process each token
	for _, tokenInfo := range tokens {
		// Convert slices/structs to JSON strings
		tagsJSON, err := json.Marshal(tokenInfo.Tags)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal tags for token %s: %w", tokenInfo.Address, err)
		}

		var dailyJSON, extensionsJSON string
		if tokenInfo.DailyVolume != nil {
			dailyBytes, err := json.Marshal(map[string]interface{}{
				"volume": *tokenInfo.DailyVolume,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to marshal daily for token %s: %w", tokenInfo.Address, err)
			}
			dailyJSON = string(dailyBytes)
		}

		if tokenInfo.Extensions != nil {
			extensionsBytes, err := json.Marshal(tokenInfo.Extensions)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal extensions for token %s: %w", tokenInfo.Address, err)
			}
			extensionsJSON = string(extensionsBytes)
		}

		// Execute the prepared statement for this token
		var token model.Token
		err = stmt.QueryRowxContext(ctx,
			tokenInfo.Address,
			1, // Default chain ID for Solana
			tokenInfo.Decimals,
			tokenInfo.Name,
			tokenInfo.Symbol,
			tokenInfo.LogoURI,
			string(tagsJSON),
			dailyJSON,
			extensionsJSON,
			now,
			now,
		).StructScan(&token)

		if err != nil {
			return nil, fmt.Errorf("failed to save token %s: %w", tokenInfo.Address, err)
		}

		result = append(result, token)
	}

	// Commit the transaction
	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return result, nil
}

////////////////////////////////////////////////////////////////////////////////
