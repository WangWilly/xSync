-- Drop token tables and their indexes
DROP INDEX IF EXISTS idx_token_embeddings_chroma_document_id;
DROP INDEX IF EXISTS idx_token_embeddings_token_address;
DROP INDEX IF EXISTS idx_tokens_chroma_embedded;
DROP INDEX IF EXISTS idx_tokens_name;
DROP INDEX IF EXISTS idx_tokens_symbol;
DROP INDEX IF EXISTS idx_tokens_address;

DROP TABLE IF EXISTS token_embeddings;
DROP TABLE IF EXISTS tokens;
