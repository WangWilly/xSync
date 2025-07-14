package embedding

import (
	"time"
)

// Config holds configuration for the embedding client
type Config struct {
	// ChromaDB settings
	ChromaDBURL    string `json:"chroma_db_url" yaml:"chroma_db_url"`
	ChromaDBToken  string `json:"chroma_db_token" yaml:"chroma_db_token"`
	CollectionName string `json:"collection_name" yaml:"collection_name"`

	// Redis settings (optional)
	RedisURL      string `json:"redis_url" yaml:"redis_url"`
	RedisPassword string `json:"redis_password" yaml:"redis_password"`
	RedisDB       int    `json:"redis_db" yaml:"redis_db"`

	// Embedding settings
	EmbeddingModel      string  `json:"embedding_model" yaml:"embedding_model"`
	EmbeddingDim        int     `json:"embedding_dim" yaml:"embedding_dim"`
	SimilarityThreshold float32 `json:"similarity_threshold" yaml:"similarity_threshold"`

	// Processing settings
	BatchSize       int           `json:"batch_size" yaml:"batch_size"`
	RequestTimeout  time.Duration `json:"request_timeout" yaml:"request_timeout"`
	CacheExpiration time.Duration `json:"cache_expiration" yaml:"cache_expiration"`
	MaxRetries      int           `json:"max_retries" yaml:"max_retries"`
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		ChromaDBURL:         "http://localhost:8000",
		ChromaDBToken:       "xsync-dev-token-2025",
		CollectionName:      "xsync_tweets",
		RedisURL:            "localhost:6379",
		RedisPassword:       "xsync-redis-2025",
		RedisDB:             0,
		EmbeddingModel:      "sentence-transformers/all-MiniLM-L6-v2",
		EmbeddingDim:        384,
		SimilarityThreshold: 0.7,
		BatchSize:           100,
		RequestTimeout:      30 * time.Second,
		CacheExpiration:     24 * time.Hour,
		MaxRetries:          3,
	}
}

// TweetEmbedding represents a tweet with its embedding and metadata
type TweetEmbedding struct {
	ID         string                 `json:"id"`
	TweetID    uint64                 `json:"tweet_id"`
	UserID     uint64                 `json:"user_id"`
	Content    string                 `json:"content"`
	UserName   string                 `json:"user_name"`
	ScreenName string                 `json:"screen_name"`
	TweetTime  time.Time              `json:"tweet_time"`
	Embedding  []float32              `json:"embedding,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
}

// SearchResult represents a search result with similarity score
type SearchResult struct {
	Tweet    *TweetEmbedding `json:"tweet"`
	Score    float32         `json:"score"`
	Distance float32         `json:"distance"`
	Rank     int             `json:"rank"`
}

// SearchQuery represents a search query with filters
type SearchQuery struct {
	Query           string            `json:"query"`
	Limit           int               `json:"limit"`
	MinScore        float32           `json:"min_score"`
	UserFilters     []uint64          `json:"user_filters,omitempty"`
	DateRange       *DateRange        `json:"date_range,omitempty"`
	TokenFilters    []string          `json:"token_filters,omitempty"`
	MetadataFilters map[string]string `json:"metadata_filters,omitempty"`
}

// DateRange represents a time range filter
type DateRange struct {
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
}

// BatchIndexRequest represents a batch indexing request
type BatchIndexRequest struct {
	Tweets    []*TweetEmbedding `json:"tweets"`
	BatchID   string            `json:"batch_id"`
	Overwrite bool              `json:"overwrite"`
}

// IndexStats represents indexing statistics
type IndexStats struct {
	TotalTweets    int64     `json:"total_tweets"`
	IndexedTweets  int64     `json:"indexed_tweets"`
	FailedTweets   int64     `json:"failed_tweets"`
	LastIndexedAt  time.Time `json:"last_indexed_at"`
	AverageScore   float32   `json:"average_score"`
	CollectionSize int64     `json:"collection_size"`
	EmbeddingModel string    `json:"embedding_model"`
}

// TokenMention represents a detected web3 token mention
type TokenMention struct {
	Token      string  `json:"token"`
	Symbol     string  `json:"symbol"`
	Confidence float32 `json:"confidence"`
	Context    string  `json:"context"`
	Position   int     `json:"position"`
	Length     int     `json:"length"`
	Category   string  `json:"category"` // DeFi, NFT, Gaming, etc.
}

// Web3Analysis represents analysis of web3 content in tweets
type Web3Analysis struct {
	TweetID        uint64          `json:"tweet_id"`
	TokenMentions  []*TokenMention `json:"token_mentions"`
	Sentiment      string          `json:"sentiment"`       // positive, negative, neutral
	SentimentScore float32         `json:"sentiment_score"` // -1.0 to 1.0
	Topics         []string        `json:"topics"`
	TrendScore     float32         `json:"trend_score"`
	InfluenceScore float32         `json:"influence_score"`
	AnalyzedAt     time.Time       `json:"analyzed_at"`
}
