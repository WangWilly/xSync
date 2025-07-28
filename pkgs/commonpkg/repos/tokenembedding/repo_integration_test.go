package tokenembedding

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver

	"github.com/WangWilly/xSync/migration/automigrate"
	"github.com/WangWilly/xSync/pkgs/commonpkg/model"
	"github.com/jmoiron/sqlx"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var db *sqlx.DB

func TestMain(m *testing.M) {
	// Setup
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	// Start a PostgreSQL container
	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "postgres",
		Tag:        "14",
		Env: []string{
			"POSTGRES_PASSWORD=postgres",
			"POSTGRES_USER=postgres",
			"POSTGRES_DB=testdb",
		},
	}, func(config *docker.HostConfig) {
		// Set AutoRemove to true so that the container is removed when stopped
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})
	if err != nil {
		log.Fatalf("Could not start resource: %s", err)
	}

	// Exponential backoff-retry until the database is ready
	if err := pool.Retry(func() error {
		var err error
		db, err = sqlx.Connect("postgres", fmt.Sprintf("postgres://postgres:postgres@localhost:%s/testdb?sslmode=disable", resource.GetPort("5432/tcp")))
		if err != nil {
			return err
		}
		return db.Ping()
	}); err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	// Set up database schema using auto migration
	err = automigrate.AutoMigrateUp(automigrate.AutoMigrateConfig{
		SqlxDB: db,
	})
	if err != nil {
		log.Fatalf("Could not run auto migration: %s", err)
	}

	// Run tests
	code := m.Run()

	// Clean up
	if err := pool.Purge(resource); err != nil {
		log.Fatalf("Could not purge resource: %s", err)
	}

	os.Exit(code)
}

func setupTestTokens() {
	// Insert test tokens that will be referenced by token embeddings
	query := `
		INSERT INTO tokens (address, chain_id, decimals, name, symbol) 
		VALUES 
			('0x123456789', 1, 18, 'Test Token 1', 'TT1'),
			('0x987654321', 1, 18, 'Test Token 2', 'TT2'),
			('0xabcdef123', 1, 18, 'Test Token 3', 'TT3')
		ON CONFLICT (address) DO NOTHING
	`
	_, err := db.Exec(query)
	if err != nil {
		log.Printf("Warning: Failed to setup test tokens: %v", err)
	}
}

func TestRepoIntegration_Upsert(t *testing.T) {
	repo := New()
	ctx := context.Background()

	// Setup test tokens
	setupTestTokens()

	t.Run("insert new token embedding", func(t *testing.T) {
		// Arrange
		tokenAddress := "0x123456789"
		documentID := "doc-123"
		content := "embedding content"

		// Act
		err := repo.Upsert(ctx, db, tokenAddress, documentID, content)

		// Assert
		require.NoError(t, err)

		// Verify insertion
		var result struct {
			ID           int64     `db:"id"`
			TokenAddress string    `db:"token_address"`
			DocumentID   string    `db:"chroma_document_id"`
			Content      string    `db:"embedding_content"`
			CreatedAt    time.Time `db:"created_at"`
			UpdatedAt    time.Time `db:"updated_at"`
		}

		err = db.Get(&result, "SELECT token_address, chroma_document_id, embedding_content, created_at, updated_at FROM token_embeddings WHERE chroma_document_id = $1", documentID)
		require.NoError(t, err)
		assert.Equal(t, tokenAddress, result.TokenAddress)
		assert.Equal(t, documentID, result.DocumentID)
		assert.Equal(t, content, result.Content)
		assert.NotZero(t, result.CreatedAt)
		assert.NotZero(t, result.UpdatedAt)
	})

	t.Run("update existing token embedding", func(t *testing.T) {
		// Arrange
		tokenAddress := "0x123456789"
		documentID := "doc-123"
		updatedContent := "updated embedding content"

		// Get current timestamp for comparison
		var beforeUpdate time.Time
		err := db.Get(&beforeUpdate, "SELECT updated_at FROM token_embeddings WHERE chroma_document_id = $1", documentID)
		require.NoError(t, err)

		// Wait a bit to ensure timestamp changes
		time.Sleep(10 * time.Millisecond)

		// Act
		err = repo.Upsert(ctx, db, tokenAddress, documentID, updatedContent)

		// Assert
		require.NoError(t, err)

		// Verify update
		var result struct {
			TokenAddress string    `db:"token_address"`
			DocumentID   string    `db:"chroma_document_id"`
			Content      string    `db:"embedding_content"`
			CreatedAt    time.Time `db:"created_at"`
			UpdatedAt    time.Time `db:"updated_at"`
		}

		err = db.Get(&result, "SELECT token_address, chroma_document_id, embedding_content, created_at, updated_at FROM token_embeddings WHERE chroma_document_id = $1", documentID)
		require.NoError(t, err)
		assert.Equal(t, tokenAddress, result.TokenAddress)
		assert.Equal(t, documentID, result.DocumentID)
		assert.Equal(t, updatedContent, result.Content)
		assert.True(t, result.UpdatedAt.After(beforeUpdate))
	})
}

func TestRepoIntegration_BatchUpsert(t *testing.T) {
	repo := New()
	ctx := context.Background()

	// Setup test tokens
	setupTestTokens()

	t.Run("batch insert new token embeddings", func(t *testing.T) {
		// Arrange
		embeddings := []*model.TokenEmbedding{
			{
				TokenAddress:     "0x123456789",
				ChromaDocumentID: "doc-batch-1",
				EmbeddingContent: "batch content 1",
			},
			{
				TokenAddress:     "0x987654321",
				ChromaDocumentID: "doc-batch-2",
				EmbeddingContent: "batch content 2",
			},
		}

		// Act
		err := repo.BatchUpsert(ctx, db, embeddings)

		// Assert
		require.NoError(t, err)

		// Verify the ID and timestamps are updated in the embeddings structs
		assert.NotZero(t, embeddings[0].CreatedAt)
		assert.NotZero(t, embeddings[0].UpdatedAt)
		assert.NotZero(t, embeddings[1].CreatedAt)
		assert.NotZero(t, embeddings[1].UpdatedAt)

		// Verify insertions in the database
		var count int
		err = db.Get(&count, "SELECT COUNT(*) FROM token_embeddings WHERE chroma_document_id IN ($1, $2)",
			embeddings[0].ChromaDocumentID, embeddings[1].ChromaDocumentID)
		require.NoError(t, err)
		assert.Equal(t, 2, count)
	})

	t.Run("batch update existing token embeddings", func(t *testing.T) {
		// Arrange - reuse the same tokens and documents from the previous test but update content
		embeddings := []*model.TokenEmbedding{
			{
				TokenAddress:     "0x123456789",
				ChromaDocumentID: "doc-batch-1",
				EmbeddingContent: "updated batch content 1",
			},
			{
				TokenAddress:     "0x987654321",
				ChromaDocumentID: "doc-batch-2",
				EmbeddingContent: "updated batch content 2",
			},
		}

		// Get current timestamps for comparison
		var beforeUpdates []time.Time
		err := db.Select(&beforeUpdates, "SELECT updated_at FROM token_embeddings WHERE chroma_document_id IN ($1, $2) ORDER BY chroma_document_id",
			embeddings[0].ChromaDocumentID, embeddings[1].ChromaDocumentID)
		require.NoError(t, err)
		require.Len(t, beforeUpdates, 2)

		// Wait a bit to ensure timestamp changes
		time.Sleep(10 * time.Millisecond)

		// Act
		err = repo.BatchUpsert(ctx, db, embeddings)

		// Assert
		require.NoError(t, err)

		// Verify updates
		var results []*model.TokenEmbedding
		err = db.Select(&results, "SELECT token_address, chroma_document_id, embedding_content, created_at, updated_at FROM token_embeddings WHERE chroma_document_id IN ($1, $2) ORDER BY chroma_document_id",
			embeddings[0].ChromaDocumentID, embeddings[1].ChromaDocumentID)
		require.NoError(t, err)
		require.Len(t, results, 2)

		assert.Equal(t, "updated batch content 1", results[0].EmbeddingContent)
		assert.Equal(t, "updated batch content 2", results[1].EmbeddingContent)
		assert.True(t, results[0].UpdatedAt.After(beforeUpdates[0]))
		assert.True(t, results[1].UpdatedAt.After(beforeUpdates[1]))
	})

	t.Run("batch upsert with empty slice", func(t *testing.T) {
		// Act
		err := repo.BatchUpsert(ctx, db, []*model.TokenEmbedding{})

		// Assert
		require.NoError(t, err)
	})
}
