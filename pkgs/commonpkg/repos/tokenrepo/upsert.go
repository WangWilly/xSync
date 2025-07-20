package tokenrepo

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/WangWilly/xSync/pkgs/commonpkg/clients/juptokenclient"
	"github.com/WangWilly/xSync/pkgs/commonpkg/model"
	"github.com/jmoiron/sqlx"
)

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
		INSERT INTO tokens (address, chain_id, decimals, name, symbol, logo_uri, tags, daily, extensions)
		VALUES (:address, :chain_id, :decimals, :name, :symbol, :logo_uri, :tags, :daily, :extensions)
		ON CONFLICT (address) DO UPDATE SET
			chain_id=:chain_id,
			decimals=:decimals,
			name=:name,
			symbol=:symbol,
			logo_uri=:logo_uri,
			tags=:tags,
			daily=:daily,
			extensions=:extensions,
			updated_at=CURRENT_TIMESTAMP
		RETURNING id, address, chain_id, decimals, name, symbol, logo_uri, tags, daily, extensions, created_at, updated_at, chroma_embedded, chroma_document_id
	`
	stmt, err := tx.PrepareNamedContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	result := make([]model.Token, len(tokens))
	for i, tokenInfo := range tokens {
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
		result[i] = model.Token{
			Address:    tokenInfo.Address,
			ChainID:    1,
			Decimals:   tokenInfo.Decimals,
			Name:       tokenInfo.Name,
			Symbol:     tokenInfo.Symbol,
			LogoURI:    tokenInfo.LogoURI,
			Tags:       string(tagsJSON),
			Daily:      dailyJSON,
			Extensions: extensionsJSON,
		}
		if err := stmt.QueryRowxContext(ctx, &result[i]).StructScan(&result[i]); err != nil {
			return nil, fmt.Errorf("failed to save token %s: %w", tokenInfo.Address, err)
		}
	}

	// Commit the transaction
	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return result, nil
}

////////////////////////////////////////////////////////////////////////////////
