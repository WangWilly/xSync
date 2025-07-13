package embedding

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/jmoiron/sqlx"
)

// SimpleClient provides a simplified ChromaDB integration without external dependencies
type SimpleClient struct {
	config    *Config
	db        *sqlx.DB
	processor *TweetProcessor
	ctx       context.Context
}

// NewSimpleClient creates a new simplified embedding client
func NewSimpleClient(db *sqlx.DB, config *Config) (*SimpleClient, error) {
	if config == nil {
		config = DefaultConfig()
	}

	ctx := context.Background()
	processor := NewTweetProcessor(config)

	return &SimpleClient{
		config:    config,
		db:        db,
		processor: processor,
		ctx:       ctx,
	}, nil
}

// Close closes the client connections
func (c *SimpleClient) Close() error {
	// No cleanup needed for simplified client
	return nil
}

// IndexTweets processes and stores tweets with embeddings
// This is a simplified version that stores embeddings in the local database
func (c *SimpleClient) IndexTweets(tweets []*TweetEmbedding) error {
	if len(tweets) == 0 {
		return nil
	}

	// In a real implementation, this would send to ChromaDB
	// For now, we'll log the embedding information
	log.Printf("Processing %d tweet embeddings", len(tweets))

	for _, tweet := range tweets {
		// Store embedding metadata (simplified)
		if err := c.storeEmbeddingMetadata(tweet); err != nil {
			log.Printf("Error storing embedding metadata for tweet %d: %v", tweet.TweetID, err)
		}
	}

	return nil
}

// storeEmbeddingMetadata stores embedding metadata in the database
func (c *SimpleClient) storeEmbeddingMetadata(tweet *TweetEmbedding) error {
	// Create embeddings table if it doesn't exist
	createTableQuery := `
	CREATE TABLE IF NOT EXISTS tweet_embeddings (
		id TEXT PRIMARY KEY,
		tweet_id INTEGER NOT NULL,
		user_id INTEGER NOT NULL,
		content TEXT NOT NULL,
		user_name TEXT,
		screen_name TEXT,
		tweet_time DATETIME,
		embedding_dim INTEGER,
		metadata TEXT,
		is_web3 BOOLEAN DEFAULT FALSE,
		web3_score REAL DEFAULT 0.0,
		sentiment TEXT,
		sentiment_score REAL DEFAULT 0.0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (tweet_id) REFERENCES tweets (tweet_id),
		FOREIGN KEY (user_id) REFERENCES users (id)
	)`

	if _, err := c.db.Exec(createTableQuery); err != nil {
		return fmt.Errorf("failed to create embeddings table: %w", err)
	}

	// Serialize metadata
	metadataJSON, err := json.Marshal(tweet.Metadata)
	if err != nil {
		return fmt.Errorf("failed to serialize metadata: %w", err)
	}

	// Extract web3 and sentiment info from metadata
	isWeb3 := false
	web3Score := 0.0
	sentiment := "neutral"
	sentimentScore := 0.0

	if tweet.Metadata != nil {
		if val, ok := tweet.Metadata["is_web3"].(bool); ok {
			isWeb3 = val
		}
		if val, ok := tweet.Metadata["web3_score"].(float32); ok {
			web3Score = float64(val)
		}
		if val, ok := tweet.Metadata["sentiment"].(string); ok {
			sentiment = val
		}
		if val, ok := tweet.Metadata["sentiment_score"].(float32); ok {
			sentimentScore = float64(val)
		}
	}

	// Insert or update embedding metadata
	query := `
	INSERT OR REPLACE INTO tweet_embeddings (
		id, tweet_id, user_id, content, user_name, screen_name, 
		tweet_time, embedding_dim, metadata, is_web3, web3_score, 
		sentiment, sentiment_score, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)`

	_, err = c.db.Exec(query,
		tweet.ID,
		tweet.TweetID,
		tweet.UserID,
		tweet.Content,
		tweet.UserName,
		tweet.ScreenName,
		tweet.TweetTime,
		len(tweet.Embedding),
		string(metadataJSON),
		isWeb3,
		web3Score,
		sentiment,
		sentimentScore,
	)

	return err
}

// Search performs a simple text-based search (without semantic search for now)
func (c *SimpleClient) Search(query *SearchQuery) ([]*SearchResult, error) {
	// Build SQL query with filters
	sqlQuery := `
	SELECT id, tweet_id, user_id, content, user_name, screen_name, 
		   tweet_time, metadata, is_web3, web3_score, sentiment, sentiment_score
	FROM tweet_embeddings 
	WHERE content LIKE ?`

	args := []interface{}{"%" + query.Query + "%"}

	// Add user filters
	if len(query.UserFilters) > 0 {
		placeholders := make([]string, len(query.UserFilters))
		for i, userID := range query.UserFilters {
			placeholders[i] = "?"
			args = append(args, userID)
		}
		sqlQuery += " AND user_id IN (" + fmt.Sprintf("%s", placeholders[0])
		for i := 1; i < len(placeholders); i++ {
			sqlQuery += ", " + placeholders[i]
		}
		sqlQuery += ")"
	}

	// Add web3 filter if specified
	if query.MetadataFilters != nil {
		if isWeb3, ok := query.MetadataFilters["is_web3"]; ok && isWeb3 == "true" {
			sqlQuery += " AND is_web3 = true"
		}
	}

	// Add date range filter
	if query.DateRange != nil {
		sqlQuery += " AND tweet_time BETWEEN ? AND ?"
		args = append(args, query.DateRange.StartTime, query.DateRange.EndTime)
	}

	// Add ordering and limit
	sqlQuery += " ORDER BY web3_score DESC, sentiment_score DESC"
	if query.Limit > 0 {
		sqlQuery += " LIMIT ?"
		args = append(args, query.Limit)
	}

	rows, err := c.db.Query(sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute search query: %w", err)
	}
	defer rows.Close()

	var results []*SearchResult
	rank := 1

	for rows.Next() {
		var tweet TweetEmbedding
		var metadataJSON string
		var isWeb3 bool
		var web3Score, sentimentScore float64
		var sentiment string

		err := rows.Scan(
			&tweet.ID,
			&tweet.TweetID,
			&tweet.UserID,
			&tweet.Content,
			&tweet.UserName,
			&tweet.ScreenName,
			&tweet.TweetTime,
			&metadataJSON,
			&isWeb3,
			&web3Score,
			&sentiment,
			&sentimentScore,
		)
		if err != nil {
			continue
		}

		// Deserialize metadata
		if err := json.Unmarshal([]byte(metadataJSON), &tweet.Metadata); err != nil {
			tweet.Metadata = make(map[string]interface{})
		}

		// Calculate simple relevance score (placeholder for semantic similarity)
		score := float32(0.5) // Base score
		if isWeb3 {
			score += float32(web3Score) * 0.3
		}
		if sentimentScore > 0 {
			score += float32(sentimentScore) * 0.2
		}

		if score < query.MinScore {
			continue
		}

		result := &SearchResult{
			Tweet:    &tweet,
			Score:    score,
			Distance: 1.0 - score,
			Rank:     rank,
		}

		results = append(results, result)
		rank++
	}

	return results, nil
}

// GetStats returns simple indexing statistics
func (c *SimpleClient) GetStats() (*IndexStats, error) {
	var totalTweets, indexedTweets, web3Tweets int64
	var avgWeb3Score float64

	// Count total tweets in embeddings table
	err := c.db.Get(&indexedTweets, "SELECT COUNT(*) FROM tweet_embeddings")
	if err != nil {
		indexedTweets = 0
	}

	// Count total tweets in main table
	err = c.db.Get(&totalTweets, "SELECT COUNT(*) FROM tweets")
	if err != nil {
		totalTweets = 0
	}

	// Count web3 tweets
	err = c.db.Get(&web3Tweets, "SELECT COUNT(*) FROM tweet_embeddings WHERE is_web3 = true")
	if err != nil {
		web3Tweets = 0
	}

	// Get average web3 score
	err = c.db.Get(&avgWeb3Score, "SELECT AVG(web3_score) FROM tweet_embeddings WHERE is_web3 = true")
	if err != nil {
		avgWeb3Score = 0
	}

	return &IndexStats{
		TotalTweets:    totalTweets,
		IndexedTweets:  indexedTweets,
		FailedTweets:   totalTweets - indexedTweets,
		LastIndexedAt:  time.Now(),
		AverageScore:   float32(avgWeb3Score),
		CollectionSize: indexedTweets,
		EmbeddingModel: c.config.EmbeddingModel,
	}, nil
}

// SearchTokenMentions searches for specific token mentions
func (c *SimpleClient) SearchTokenMentions(tokenQuery string) ([]*SearchResult, error) {
	query := &SearchQuery{
		Query:    tokenQuery,
		Limit:    100,
		MinScore: c.config.SimilarityThreshold,
		MetadataFilters: map[string]string{
			"is_web3": "true",
		},
	}

	return c.Search(query)
}
