package tokenrepo

import (
	"context"
	"fmt"
	"log"
	"testing"

	"github.com/WangWilly/xSync/pkgs/commonpkg/clients/juptokenclient"
	"github.com/WangWilly/xSync/pkgs/commonpkg/model"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var db *sqlx.DB

func TestMain(m *testing.M) {
	// uses a sensible default on windows (tcp/http) and linux/osx (socket)
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	// pulls an image, creates a container based on it and runs it
	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "postgres",
		Tag:        "14",
		Env: []string{
			"POSTGRES_PASSWORD=postgres",
			"POSTGRES_USER=postgres",
			"POSTGRES_DB=testdb",
		},
	}, func(config *docker.HostConfig) {
		// set AutoRemove to true so that stopped container goes away by itself
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{
			Name: "no",
		}
	})
	if err != nil {
		log.Fatalf("Could not start resource: %s", err)
	}

	// exponential backoff-retry, because the application in the container might not be ready to accept connections yet
	if err := pool.Retry(func() error {
		var err error
		db, err = sqlx.Open("postgres", fmt.Sprintf("postgres://postgres:postgres@localhost:%s/testdb?sslmode=disable", resource.GetPort("5432/tcp")))
		if err != nil {
			return err
		}
		return db.Ping()
	}); err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	// Create the tokens table
	setupSchema()

	// Run the tests
	code := m.Run()

	// You can't defer this because os.Exit doesn't care for defer
	if err := pool.Purge(resource); err != nil {
		log.Fatalf("Could not purge resource: %s", err)
	}

	// Exit with the status code from the tests
	log.Printf("Tests finished with exit code %d", code)
}

func setupSchema() {
	// Create tokens table
	schema := `
	CREATE TABLE IF NOT EXISTS tokens (
		id SERIAL PRIMARY KEY,
		address TEXT NOT NULL UNIQUE,
		chain_id INTEGER NOT NULL,
		decimals INTEGER NOT NULL,
		name TEXT NOT NULL,
		symbol TEXT NOT NULL,
		logo_uri TEXT,
		tags TEXT,
		daily TEXT,
		extensions TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		chroma_embedded BOOLEAN DEFAULT false,
		chroma_document_id TEXT
	);
	CREATE INDEX IF NOT EXISTS tokens_address_idx ON tokens(address);
	`

	_, err := db.Exec(schema)
	if err != nil {
		log.Fatalf("Could not create schema: %s", err)
	}
}

func clearData() {
	_, err := db.Exec("TRUNCATE TABLE tokens RESTART IDENTITY")
	if err != nil {
		log.Printf("Warning: Failed to clear data: %v", err)
	}
}

func TestRepoIntegration_BatchUpsertFromJupTokenDto(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	repo := New()
	ctx := context.Background()

	// Clear data before tests
	clearData()

	t.Run("empty tokens array", func(t *testing.T) {
		tokens, err := repo.BatchUpsertFromJupTokenDto(ctx, db, []juptokenclient.JupTokenDto{})
		assert.NoError(t, err)
		assert.Empty(t, tokens)
	})

	t.Run("batch inserting new tokens", func(t *testing.T) {
		// Create test tokens
		dailyVolume1 := float64(1000000)
		dailyVolume2 := float64(2000000)

		tokenBatch := []juptokenclient.JupTokenDto{
			{
				Address:     "BatchToken1",
				Decimals:    9,
				Name:        "Batch Token 1",
				Symbol:      "BT1",
				LogoURI:     "https://example.com/logo1.png",
				Tags:        []string{"batch", "token1"},
				DailyVolume: &dailyVolume1,
				Extensions:  map[string]string{"key": "value1"},
			},
			{
				Address:     "BatchToken2",
				Decimals:    18,
				Name:        "Batch Token 2",
				Symbol:      "BT2",
				LogoURI:     "https://example.com/logo2.png",
				Tags:        []string{"batch", "token2"},
				DailyVolume: &dailyVolume2,
				Extensions:  map[string]string{"key": "value2"},
			},
		}

		// Insert the tokens
		tokens, err := repo.BatchUpsertFromJupTokenDto(ctx, db, tokenBatch)
		require.NoError(t, err)
		require.Len(t, tokens, 2)

		// Verify first token
		assert.Equal(t, tokenBatch[0].Address, tokens[0].Address)
		assert.Equal(t, tokenBatch[0].Name, tokens[0].Name)
		assert.Equal(t, tokenBatch[0].Symbol, tokens[0].Symbol)

		// Verify second token
		assert.Equal(t, tokenBatch[1].Address, tokens[1].Address)
		assert.Equal(t, tokenBatch[1].Name, tokens[1].Name)
		assert.Equal(t, tokenBatch[1].Symbol, tokens[1].Symbol)
	})

	t.Run("batch updating existing tokens", func(t *testing.T) {
		// Update the tokens from previous test
		dailyVolume1 := float64(3000000)
		dailyVolume2 := float64(4000000)

		tokenBatch := []juptokenclient.JupTokenDto{
			{
				Address:     "BatchToken1", // Same address for update
				Decimals:    10,            // Changed
				Name:        "Updated Batch Token 1",
				Symbol:      "UBT1",
				LogoURI:     "https://example.com/updated-logo1.png",
				Tags:        []string{"updated", "batch", "token1"},
				DailyVolume: &dailyVolume1,
				Extensions:  map[string]string{"key": "updated-value1"},
			},
			{
				Address:     "BatchToken2", // Same address for update
				Decimals:    20,            // Changed
				Name:        "Updated Batch Token 2",
				Symbol:      "UBT2",
				LogoURI:     "https://example.com/updated-logo2.png",
				Tags:        []string{"updated", "batch", "token2"},
				DailyVolume: &dailyVolume2,
				Extensions:  map[string]string{"key": "updated-value2"},
			},
		}

		// Get original tokens
		var originalTokens []model.Token
		err := db.Select(&originalTokens, "SELECT * FROM tokens WHERE address IN ('BatchToken1', 'BatchToken2') ORDER BY address")
		require.NoError(t, err)
		require.Len(t, originalTokens, 2)

		// Force updated_at to be older by directly updating the database
		_, err = db.Exec("UPDATE tokens SET updated_at = updated_at - interval '1 minute' WHERE address IN ('BatchToken1', 'BatchToken2')")
		require.NoError(t, err)

		// Get the updated original tokens with older timestamps
		err = db.Select(&originalTokens, "SELECT * FROM tokens WHERE address IN ('BatchToken1', 'BatchToken2') ORDER BY address")
		require.NoError(t, err)
		require.Len(t, originalTokens, 2)

		// Update the tokens
		tokens, err := repo.BatchUpsertFromJupTokenDto(ctx, db, tokenBatch)
		require.NoError(t, err)
		require.Len(t, tokens, 2)

		// Verify first token was updated
		assert.Equal(t, tokenBatch[0].Address, tokens[0].Address)
		assert.Equal(t, tokenBatch[0].Name, tokens[0].Name)
		assert.Equal(t, tokenBatch[0].Symbol, tokens[0].Symbol)
		assert.Equal(t, tokenBatch[0].Decimals, tokens[0].Decimals)

		// Verify second token was updated
		assert.Equal(t, tokenBatch[1].Address, tokens[1].Address)
		assert.Equal(t, tokenBatch[1].Name, tokens[1].Name)
		assert.Equal(t, tokenBatch[1].Symbol, tokens[1].Symbol)
		assert.Equal(t, tokenBatch[1].Decimals, tokens[1].Decimals)

		// Check updated_at timestamps
		for _, token := range tokens {
			matchingOriginal := -1
			for j, orig := range originalTokens {
				if orig.Address == token.Address {
					matchingOriginal = j
					break
				}
			}

			require.NotEqual(t, -1, matchingOriginal, "Could not find matching original token")

			// created_at should not change
			assert.Equal(t, originalTokens[matchingOriginal].CreatedAt.Unix(), token.CreatedAt.Unix())

			// updated_at should be later
			assert.Greater(t, token.UpdatedAt.Unix(), originalTokens[matchingOriginal].UpdatedAt.Unix())
		}
	})
}

func TestRepoIntegration_TotalCount(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	repo := New()
	ctx := context.Background()

	// Clear data before test
	clearData()

	t.Run("empty table", func(t *testing.T) {
		count, err := repo.TotalCount(ctx, db)
		assert.NoError(t, err)
		assert.Equal(t, 0, count)
	})

	t.Run("with records", func(t *testing.T) {
		// Insert a few tokens
		tokens := []struct {
			Address  string
			ChainID  int
			Decimals int
			Name     string
			Symbol   string
		}{
			{Address: "Token1", ChainID: 1, Decimals: 9, Name: "Token 1", Symbol: "TK1"},
			{Address: "Token2", ChainID: 1, Decimals: 9, Name: "Token 2", Symbol: "TK2"},
			{Address: "Token3", ChainID: 1, Decimals: 9, Name: "Token 3", Symbol: "TK3"},
		}

		for _, token := range tokens {
			_, err := db.Exec(`
				INSERT INTO tokens(address, chain_id, decimals, name, symbol)
				VALUES($1, $2, $3, $4, $5)
			`, token.Address, token.ChainID, token.Decimals, token.Name, token.Symbol)
			require.NoError(t, err)
		}

		// Verify count
		count, err := repo.TotalCount(ctx, db)
		assert.NoError(t, err)
		assert.Equal(t, len(tokens), count)
	})
}
