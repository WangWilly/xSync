package database

import (
	"fmt"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	log "github.com/sirupsen/logrus"
)

// ConnectPostgres connects to PostgreSQL database
func ConnectPostgres(host, port, user, password, dbname string) (*sqlx.DB, error) {
	logger := log.WithFields(log.Fields{
		"caller": "ConnectPostgres",
		"host":   host,
		"port":   port,
		"dbname": dbname,
	})

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	logger.Info("Connected to PostgreSQL database")
	return db, nil
}

// CreateTokenTables creates the token-related tables
func CreateTokenTables(db *sqlx.DB) error {
	// Create tokens table
	tokensTable := `
	CREATE TABLE IF NOT EXISTS tokens (
		id SERIAL PRIMARY KEY,
		address VARCHAR(255) NOT NULL UNIQUE,
		chain_id INTEGER NOT NULL,
		decimals INTEGER NOT NULL,
		name VARCHAR(255) NOT NULL,
		symbol VARCHAR(100) NOT NULL,
		logo_uri TEXT,
		tags TEXT,
		daily TEXT,
		extensions TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		chroma_embedded BOOLEAN DEFAULT FALSE,
		chroma_document_id VARCHAR(255) NOT NULL
	);
	`

	// Create token_embeddings table
	embedTable := `
	CREATE TABLE IF NOT EXISTS token_embeddings (
		id SERIAL PRIMARY KEY,
		token_address VARCHAR(255) NOT NULL REFERENCES tokens(address),
		chroma_document_id VARCHAR(255) NOT NULL UNIQUE,
		embedding_content TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	`

	// Create indexes
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_tokens_address ON tokens(address);",
		"CREATE INDEX IF NOT EXISTS idx_tokens_symbol ON tokens(symbol);",
		"CREATE INDEX IF NOT EXISTS idx_tokens_name ON tokens(name);",
		"CREATE INDEX IF NOT EXISTS idx_tokens_chroma_embedded ON tokens(chroma_embedded);",
		"CREATE INDEX IF NOT EXISTS idx_token_embeddings_token_address ON token_embeddings(token_address);",
		"CREATE INDEX IF NOT EXISTS idx_token_embeddings_chroma_document_id ON token_embeddings(chroma_document_id);",
	}

	// Execute table creation
	for _, query := range []string{tokensTable, embedTable} {
		if _, err := db.Exec(query); err != nil {
			return fmt.Errorf("failed to create table: %w", err)
		}
	}

	// Create indexes
	for _, index := range indexes {
		if _, err := db.Exec(index); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	log.Info("Token tables created successfully")
	return nil
}
