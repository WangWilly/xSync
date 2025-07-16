package chromatokenclient

import (
	"context"
	"fmt"
	"strings"

	"github.com/WangWilly/xSync/pkgs/commonpkg/clients/juptokenclient"
	chroma "github.com/amikos-tech/chroma-go/pkg/api/v2"
	"github.com/google/uuid"
)

func (c *ChromaTokenClient) CreateTokenFromJupiter(ctx context.Context, tokenInfo *juptokenclient.JupTokenDto) (string, error) {
	docID := uuid.New().String()
	metadata := c.createTokenMetaFromJupiter(tokenInfo)
	docText := c.createTokenDocumentFromJupiter(tokenInfo)

	// Add to collection
	err := c.collection.Add(ctx,
		chroma.WithIDs(chroma.DocumentID(docID)),
		chroma.WithTexts(docText),
		chroma.WithMetadatas(metadata),
	)
	if err != nil {
		return "", fmt.Errorf("failed to add token to chroma: %w", err)
	}

	return docID, nil
}

func (c *ChromaTokenClient) BatchCreateTokensFromJupiter(ctx context.Context, tokens []juptokenclient.JupTokenDto) ([]string, error) {
	var docIDs []chroma.DocumentID
	var docTexts []string
	var metadatas []chroma.DocumentMetadata

	for _, token := range tokens {
		docIDs = append(docIDs, chroma.DocumentID(uuid.New().String()))
		docTexts = append(docTexts, c.createTokenDocumentFromJupiter(&token))
		metadatas = append(metadatas, c.createTokenMetaFromJupiter(&token))
	}

	// Add to collection in batch
	err := c.collection.Add(ctx,
		chroma.WithIDs(docIDs...),
		chroma.WithTexts(docTexts...),
		chroma.WithMetadatas(metadatas...),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to add tokens to chroma: %w", err)
	}

	// Convert DocumentID to string for return
	var ids []string
	for _, id := range docIDs {
		ids = append(ids, string(id))
	}
	return ids, nil
}

////////////////////////////////////////////////////////////////////////////////

func (c *ChromaTokenClient) createTokenMetaFromJupiter(tokenInfo *juptokenclient.JupTokenDto) chroma.DocumentMetadata {
	metadataAttrs := []*chroma.MetaAttribute{
		chroma.NewStringAttribute("address", tokenInfo.Address),
		chroma.NewStringAttribute("name", tokenInfo.Name),
		chroma.NewStringAttribute("symbol", tokenInfo.Symbol),
		chroma.NewIntAttribute("chain_id", int64(1)), // Default to Solana
		chroma.NewIntAttribute("decimals", int64(tokenInfo.Decimals)),
		chroma.NewStringAttribute("logo_uri", tokenInfo.LogoURI),
	}

	// Add tags if they exist
	if len(tokenInfo.Tags) > 0 {
		metadataAttrs = append(metadataAttrs, chroma.NewStringAttribute("tags", strings.Join(tokenInfo.Tags, ",")))
	}

	// Add daily volume if it exists
	if tokenInfo.DailyVolume != nil {
		metadataAttrs = append(metadataAttrs, chroma.NewFloatAttribute("daily_volume", float64(*tokenInfo.DailyVolume)))
	}

	// Add extensions if they exist
	if tokenInfo.Extensions != nil {
		if coingeckoID, exists := tokenInfo.Extensions["coingecko_id"]; exists && coingeckoID != "" {
			metadataAttrs = append(metadataAttrs, chroma.NewStringAttribute("coingecko_id", coingeckoID))
		}
		if description, exists := tokenInfo.Extensions["description"]; exists && description != "" {
			metadataAttrs = append(metadataAttrs, chroma.NewStringAttribute("description", description))
		}
		if website, exists := tokenInfo.Extensions["website"]; exists && website != "" {
			metadataAttrs = append(metadataAttrs, chroma.NewStringAttribute("website", website))
		}
		if twitter, exists := tokenInfo.Extensions["twitter"]; exists && twitter != "" {
			metadataAttrs = append(metadataAttrs, chroma.NewStringAttribute("twitter", twitter))
		}
	}

	return chroma.NewDocumentMetadata(metadataAttrs...)
}

func (c *ChromaTokenClient) createTokenDocumentFromJupiter(tokenInfo *juptokenclient.JupTokenDto) string {
	var parts []string

	// Add basic info
	parts = append(parts, fmt.Sprintf("Token: %s", tokenInfo.Name))
	parts = append(parts, fmt.Sprintf("Symbol: %s", tokenInfo.Symbol))
	parts = append(parts, fmt.Sprintf("Address: %s", tokenInfo.Address))

	// Add tags
	if len(tokenInfo.Tags) > 0 {
		parts = append(parts, fmt.Sprintf("Tags: %s", strings.Join(tokenInfo.Tags, ", ")))
	}

	// Add description from extensions
	if tokenInfo.Extensions != nil {
		if description, exists := tokenInfo.Extensions["description"]; exists && description != "" {
			parts = append(parts, fmt.Sprintf("Description: %s", description))
		}
	}

	// Add website if available
	if tokenInfo.Extensions != nil {
		if website, exists := tokenInfo.Extensions["website"]; exists && website != "" {
			parts = append(parts, fmt.Sprintf("Website: %s", website))
		}
	}

	return strings.Join(parts, " | ")
}
