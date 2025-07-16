package chromatokenclient

import (
	"context"
	"fmt"

	chroma "github.com/amikos-tech/chroma-go/pkg/api/v2"
)

const (
	COLLECTION_NAME = "jupiter-tokens"
)

type ChromaTokenClient struct {
	client     chroma.Client
	collection chroma.Collection
}

// New creates a new Chroma client for token operations
func New(chromaURL string) (*ChromaTokenClient, error) {
	// Create HTTP client
	client, err := chroma.NewHTTPClient(
		chroma.WithBaseURL(chromaURL),
		chroma.WithDebug(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create chroma client: %w", err)
	}

	// Get or create collection for tokens
	collection, err := client.GetOrCreateCollection(
		context.Background(),
		COLLECTION_NAME,
		chroma.WithCollectionMetadataCreate(
			chroma.NewMetadata(
				chroma.NewStringAttribute("description", "Jupiter token collection for semantic search"),
				chroma.NewStringAttribute("type", "tokens"),
			),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get or create collection: %w", err)
	}

	return &ChromaTokenClient{
		client:     client,
		collection: collection,
	}, nil
}

// Close closes the Chroma client
func (c *ChromaTokenClient) Close() error {
	return c.client.Close()
}

////////////////////////////////////////////////////////////////////////////////

// GetCollectionCount returns the number of tokens in the collection
func (c *ChromaTokenClient) GetCollectionCount(ctx context.Context) (int, error) {
	count, err := c.collection.Count(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get collection count: %w", err)
	}

	return count, nil
}
