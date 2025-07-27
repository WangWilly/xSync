-- Add token tables for SQLite
CREATE TABLE IF NOT EXISTS tokens (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	address VARCHAR(255) NOT NULL UNIQUE,
	chain_id INTEGER NOT NULL,
	decimals INTEGER NOT NULL,
	name VARCHAR(255) NOT NULL,
	symbol VARCHAR(100) NOT NULL,
	logo_uri TEXT,
	tags TEXT,
	daily TEXT,
	extensions TEXT,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	chroma_embedded BOOLEAN DEFAULT FALSE,
	chroma_document_id VARCHAR(255)
);

CREATE TABLE IF NOT EXISTS token_embeddings (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	token_address VARCHAR(255) NOT NULL UNIQUE REFERENCES tokens(address),
	chroma_document_id VARCHAR(255) NOT NULL UNIQUE,
	embedding_content TEXT NOT NULL,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for tokens
CREATE INDEX IF NOT EXISTS idx_tokens_address ON tokens(address);
CREATE INDEX IF NOT EXISTS idx_tokens_symbol ON tokens(symbol);
CREATE INDEX IF NOT EXISTS idx_tokens_name ON tokens(name);
CREATE INDEX IF NOT EXISTS idx_tokens_chroma_embedded ON tokens(chroma_embedded);
CREATE INDEX IF NOT EXISTS idx_token_embeddings_token_address ON token_embeddings(token_address);
CREATE INDEX IF NOT EXISTS idx_token_embeddings_chroma_document_id ON token_embeddings(chroma_document_id);
