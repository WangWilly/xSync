package embedding

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/WangWilly/xSync/pkgs/database"
	"github.com/WangWilly/xSync/pkgs/model"
	"github.com/jmoiron/sqlx"
)

// IndexerService handles indexing tweets from the database to ChromaDB
type IndexerService struct {
	db        *sqlx.DB
	client    *SimpleClient
	processor *TweetProcessor
	config    *Config
}

// NewIndexerService creates a new indexer service
func NewIndexerService(db *sqlx.DB, config *Config) (*IndexerService, error) {
	client, err := NewSimpleClient(db, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create embedding client: %w", err)
	}

	processor := NewTweetProcessor(config)

	return &IndexerService{
		db:        db,
		client:    client,
		processor: processor,
		config:    config,
	}, nil
}

// Close closes the indexer service
func (is *IndexerService) Close() error {
	return is.client.Close()
}

// IndexAllTweets indexes all tweets from the database
func (is *IndexerService) IndexAllTweets() error {
	log.Println("Starting to index all tweets...")

	// Get all users first for metadata
	users, err := is.getAllUsers()
	if err != nil {
		return fmt.Errorf("failed to get users: %w", err)
	}

	userMap := make(map[uint64]*model.User)
	for _, user := range users {
		userMap[user.Id] = user
	}

	// Process tweets in batches
	offset := 0
	batchSize := is.config.BatchSize

	for {
		tweets, err := is.getTweetsBatch(offset, batchSize)
		if err != nil {
			return fmt.Errorf("failed to get tweets batch: %w", err)
		}

		if len(tweets) == 0 {
			break
		}

		log.Printf("Processing batch of %d tweets (offset %d)", len(tweets), offset)

		// Process tweets
		embeddings, err := is.processor.ProcessTweets(tweets, userMap)
		if err != nil {
			log.Printf("Error processing tweets batch: %v", err)
			offset += batchSize
			continue
		}

		// Index embeddings
		if err := is.client.IndexTweets(embeddings); err != nil {
			log.Printf("Error indexing tweets batch: %v", err)
			offset += batchSize
			continue
		}

		log.Printf("Successfully indexed %d tweets", len(embeddings))
		offset += batchSize

		// Small delay to avoid overwhelming the system
		time.Sleep(100 * time.Millisecond)
	}

	log.Println("Finished indexing all tweets")
	return nil
}

// IndexNewTweets indexes tweets that haven't been indexed yet
func (is *IndexerService) IndexNewTweets(since time.Time) error {
	log.Printf("Indexing new tweets since %v", since)

	// Get users
	users, err := is.getAllUsers()
	if err != nil {
		return fmt.Errorf("failed to get users: %w", err)
	}

	userMap := make(map[uint64]*model.User)
	for _, user := range users {
		userMap[user.Id] = user
	}

	// Get new tweets
	tweets, err := is.getNewTweets(since)
	if err != nil {
		return fmt.Errorf("failed to get new tweets: %w", err)
	}

	if len(tweets) == 0 {
		log.Println("No new tweets to index")
		return nil
	}

	log.Printf("Found %d new tweets to index", len(tweets))

	// Process in batches
	for i := 0; i < len(tweets); i += is.config.BatchSize {
		end := i + is.config.BatchSize
		if end > len(tweets) {
			end = len(tweets)
		}

		batch := tweets[i:end]
		embeddings, err := is.processor.ProcessTweets(batch, userMap)
		if err != nil {
			log.Printf("Error processing tweets batch %d-%d: %v", i, end, err)
			continue
		}

		if err := is.client.IndexTweets(embeddings); err != nil {
			log.Printf("Error indexing tweets batch %d-%d: %v", i, end, err)
			continue
		}

		log.Printf("Successfully indexed batch %d-%d (%d tweets)", i, end, len(embeddings))
	}

	return nil
}

// IndexUserTweets indexes all tweets for a specific user
func (is *IndexerService) IndexUserTweets(userID uint64) error {
	log.Printf("Indexing tweets for user %d", userID)

	// Get user
	user, err := database.GetUserById(is.db, userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	if user == nil {
		return fmt.Errorf("user %d not found", userID)
	}

	// Get user's tweets
	tweets, err := is.getUserTweets(userID)
	if err != nil {
		return fmt.Errorf("failed to get user tweets: %w", err)
	}

	if len(tweets) == 0 {
		log.Printf("No tweets found for user %d", userID)
		return nil
	}

	log.Printf("Found %d tweets for user %d", len(tweets), userID)

	// Create user map
	userMap := map[uint64]*model.User{userID: user}

	// Process tweets
	embeddings, err := is.processor.ProcessTweets(tweets, userMap)
	if err != nil {
		return fmt.Errorf("failed to process tweets: %w", err)
	}

	// Index embeddings
	if err := is.client.IndexTweets(embeddings); err != nil {
		return fmt.Errorf("failed to index tweets: %w", err)
	}

	log.Printf("Successfully indexed %d tweets for user %d", len(embeddings), userID)
	return nil
}

// SearchTweets searches for tweets using semantic search
func (is *IndexerService) SearchTweets(query string, limit int) ([]*SearchResult, error) {
	searchQuery := &SearchQuery{
		Query:    query,
		Limit:    limit,
		MinScore: is.config.SimilarityThreshold,
	}

	return is.client.Search(searchQuery)
}

// SearchWeb3Tweets searches specifically for web3-related tweets
func (is *IndexerService) SearchWeb3Tweets(query string, limit int) ([]*SearchResult, error) {
	searchQuery := &SearchQuery{
		Query:    query,
		Limit:    limit,
		MinScore: is.config.SimilarityThreshold,
		MetadataFilters: map[string]string{
			"is_web3": "true",
		},
	}

	return is.client.Search(searchQuery)
}

// GetIndexStats returns indexing statistics
func (is *IndexerService) GetIndexStats() (*IndexStats, error) {
	return is.client.GetStats()
}

// Database query helpers

func (is *IndexerService) getAllUsers() ([]*model.User, error) {
	var users []*model.User
	err := is.db.Select(&users, "SELECT * FROM users ORDER BY id")
	return users, err
}

func (is *IndexerService) getTweetsBatch(offset, limit int) ([]*model.Tweet, error) {
	var tweets []*model.Tweet
	err := is.db.Select(&tweets, "SELECT * FROM tweets ORDER BY id LIMIT ? OFFSET ?", limit, offset)
	return tweets, err
}

func (is *IndexerService) getNewTweets(since time.Time) ([]*model.Tweet, error) {
	var tweets []*model.Tweet
	err := is.db.Select(&tweets, "SELECT * FROM tweets WHERE created_at > ? ORDER BY created_at", since)
	return tweets, err
}

func (is *IndexerService) getUserTweets(userID uint64) ([]*model.Tweet, error) {
	var tweets []*model.Tweet
	err := is.db.Select(&tweets, "SELECT * FROM tweets WHERE user_id = ? ORDER BY tweet_time DESC", userID)
	return tweets, err
}

// AutoIndexer runs continuous indexing of new tweets
type AutoIndexer struct {
	indexer  *IndexerService
	interval time.Duration
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewAutoIndexer creates a new auto indexer
func NewAutoIndexer(indexer *IndexerService, interval time.Duration) *AutoIndexer {
	ctx, cancel := context.WithCancel(context.Background())
	return &AutoIndexer{
		indexer:  indexer,
		interval: interval,
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Start starts the auto indexer
func (ai *AutoIndexer) Start() {
	go ai.run()
}

// Stop stops the auto indexer
func (ai *AutoIndexer) Stop() {
	ai.cancel()
}

// run runs the auto indexing loop
func (ai *AutoIndexer) run() {
	ticker := time.NewTicker(ai.interval)
	defer ticker.Stop()

	lastIndexed := time.Now().Add(-24 * time.Hour) // Start with last 24 hours

	for {
		select {
		case <-ai.ctx.Done():
			return
		case <-ticker.C:
			log.Println("Auto-indexing new tweets...")

			if err := ai.indexer.IndexNewTweets(lastIndexed); err != nil {
				log.Printf("Auto-indexing error: %v", err)
			} else {
				lastIndexed = time.Now()
				log.Println("Auto-indexing completed")
			}
		}
	}
}
