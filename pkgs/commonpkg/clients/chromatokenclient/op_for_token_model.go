package chromatokenclient

import (
	"context"
	"fmt"
	"strings"

	"github.com/WangWilly/xSync/pkgs/commonpkg/model"
	chroma "github.com/amikos-tech/chroma-go/pkg/api/v2"
	"github.com/google/uuid"
)

// AddToken adds a token to the Chroma collection
func (c *ChromaTokenClient) AddToken(ctx context.Context, token *model.Token) (string, error) {
	// Generate unique document ID
	docID := uuid.New().String()

	// Create document text for embedding
	docText := c.createTokenDocument(token)

	// Create metadata
	metadata := chroma.NewDocumentMetadata(
		chroma.NewStringAttribute("address", token.Address),
		chroma.NewStringAttribute("name", token.Name),
		chroma.NewStringAttribute("symbol", token.Symbol),
		chroma.NewIntAttribute("chain_id", int64(token.ChainID)),
		chroma.NewIntAttribute("decimals", int64(token.Decimals)),
		chroma.NewStringAttribute("logo_uri", token.LogoURI),
		chroma.NewStringAttribute("tags", token.Tags),
		chroma.NewStringAttribute("daily", token.Daily),
		chroma.NewStringAttribute("extensions", token.Extensions),
	)

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

// createTokenDocument creates a searchable document text from token data
func (c *ChromaTokenClient) createTokenDocument(token *model.Token) string {
	var parts []string

	// Add basic info
	parts = append(parts, fmt.Sprintf("Token: %s", token.Name))
	parts = append(parts, fmt.Sprintf("Symbol: %s", token.Symbol))
	parts = append(parts, fmt.Sprintf("Address: %s", token.Address))

	// Add description from extensions if available
	if token.Extensions != "" {
		// Parse extensions JSON to extract description
		parts = append(parts, fmt.Sprintf("Extensions: %s", token.Extensions))
	}

	// Add tags if available
	if token.Tags != "" {
		parts = append(parts, fmt.Sprintf("Tags: %s", token.Tags))
	}

	return strings.Join(parts, " | ")
}
