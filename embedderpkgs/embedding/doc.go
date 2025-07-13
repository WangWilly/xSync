// Package embedding provides ChromaDB integration for xSync tweet embeddings.
//
// This package enables storing and searching tweet content using semantic embeddings
// for LLM-profitable web3 token analysis. It integrates with:
//   - ChromaDB for vector storage and similarity search
//   - SentenceTransformers for text embeddings
//   - CrossEncoder for result re-ranking
//   - Redis for caching (optional)
//
// The package supports:
//   - Tweet content embedding and storage
//   - Semantic search with similarity scoring
//   - Web3 token mention detection and analysis
//   - Batch processing for large datasets
//   - Real-time indexing of new tweets
//
// Usage:
//
//	client, err := embedding.NewClient(config)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer client.Close()
//
//	// Store tweet embeddings
//	err = client.IndexTweets(tweets)
//
//	// Search for web3 tokens
//	results, err := client.SearchTokenMentions("DeFi yield farming")
package embedding
