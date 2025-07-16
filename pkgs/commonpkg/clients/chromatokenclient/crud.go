package chromatokenclient

import (
	"context"
	"fmt"

	chroma "github.com/amikos-tech/chroma-go/pkg/api/v2"
)

// GetTokenByID retrieves a token by its Chroma document ID
func (c *ChromaTokenClient) GetTokenByID(ctx context.Context, docID string) (chroma.GetResult, error) {
	results, err := c.collection.Get(ctx,
		chroma.WithIDsGet(chroma.DocumentID(docID)),
		chroma.WithIncludeGet(chroma.IncludeMetadatas, chroma.IncludeDocuments),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get token by ID: %w", err)
	}

	return results, nil
}

// GetTokens searches for tokens using semantic search
func (c *ChromaTokenClient) GetTokens(ctx context.Context, query string, limit int) (chroma.QueryResult, error) {
	results, err := c.collection.Query(ctx,
		chroma.WithQueryTexts(query),
		chroma.WithNResults(limit),
		chroma.WithIncludeQuery(chroma.IncludeMetadatas, chroma.IncludeDocuments),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to search tokens: %w", err)
	}

	return results, nil
}

// DeleteToken removes a token from the Chroma collection
func (c *ChromaTokenClient) DeleteToken(ctx context.Context, docID string) error {
	err := c.collection.Delete(ctx,
		chroma.WithIDsDelete(chroma.DocumentID(docID)),
	)
	if err != nil {
		return fmt.Errorf("failed to delete token: %w", err)
	}

	return nil
}
