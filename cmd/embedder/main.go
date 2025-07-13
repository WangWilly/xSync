package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/WangWilly/xSync/embedderpkgs/embedding"
	"github.com/WangWilly/xSync/pkgs/database"
)

func main() {
	var (
		dbPath      = flag.String("db", "./conf/data/xSync.db", "Path to SQLite database")
		chromaURL   = flag.String("chroma-url", "http://localhost:8000", "ChromaDB URL")
		chromaToken = flag.String("chroma-token", "xsync-dev-token-2025", "ChromaDB authentication token")
		redisURL    = flag.String("redis-url", "localhost:6379", "Redis URL")
		redisPass   = flag.String("redis-pass", "xsync-redis-2025", "Redis password")
		command     = flag.String("cmd", "help", "Command to run: index-all, index-new, index-user, search, stats, auto, help")
		userID      = flag.String("user", "", "User ID for user-specific operations")
		query       = flag.String("query", "", "Search query")
		limit       = flag.Int("limit", 50, "Search result limit")
		interval    = flag.Duration("interval", 5*time.Minute, "Auto-indexing interval")
	)
	flag.Parse()

	// Create configuration
	config := &embedding.Config{
		ChromaDBURL:         *chromaURL,
		ChromaDBToken:       *chromaToken,
		CollectionName:      "xsync_tweets",
		RedisURL:            *redisURL,
		RedisPassword:       *redisPass,
		RedisDB:             0,
		EmbeddingModel:      "sentence-transformers/all-MiniLM-L6-v2",
		EmbeddingDim:        384,
		SimilarityThreshold: 0.7,
		BatchSize:           100,
		RequestTimeout:      30 * time.Second,
		CacheExpiration:     24 * time.Hour,
		MaxRetries:          3,
	}

	// Connect to database
	db, err := database.ConnectDatabase(*dbPath)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Create indexer service
	indexer, err := embedding.NewIndexerService(db, config)
	if err != nil {
		log.Fatalf("Failed to create indexer service: %v", err)
	}
	defer indexer.Close()

	// Execute command
	switch *command {
	case "index-all":
		err = indexer.IndexAllTweets()
		if err != nil {
			log.Fatalf("Failed to index all tweets: %v", err)
		}
		fmt.Println("Successfully indexed all tweets")

	case "index-new":
		since := time.Now().Add(-24 * time.Hour) // Default to last 24 hours
		err = indexer.IndexNewTweets(since)
		if err != nil {
			log.Fatalf("Failed to index new tweets: %v", err)
		}
		fmt.Println("Successfully indexed new tweets")

	case "index-user":
		if *userID == "" {
			log.Fatal("User ID is required for index-user command")
		}
		uid, err := strconv.ParseUint(*userID, 10, 64)
		if err != nil {
			log.Fatalf("Invalid user ID: %v", err)
		}
		err = indexer.IndexUserTweets(uid)
		if err != nil {
			log.Fatalf("Failed to index user tweets: %v", err)
		}
		fmt.Printf("Successfully indexed tweets for user %d\n", uid)

	case "search":
		if *query == "" {
			log.Fatal("Query is required for search command")
		}
		results, err := indexer.SearchTweets(*query, *limit)
		if err != nil {
			log.Fatalf("Failed to search tweets: %v", err)
		}
		printSearchResults(results)

	case "search-web3":
		if *query == "" {
			log.Fatal("Query is required for search-web3 command")
		}
		results, err := indexer.SearchWeb3Tweets(*query, *limit)
		if err != nil {
			log.Fatalf("Failed to search web3 tweets: %v", err)
		}
		printSearchResults(results)

	case "stats":
		stats, err := indexer.GetIndexStats()
		if err != nil {
			log.Fatalf("Failed to get stats: %v", err)
		}
		printStats(stats)

	case "auto":
		fmt.Printf("Starting auto-indexer with interval %v\n", *interval)
		autoIndexer := embedding.NewAutoIndexer(indexer, *interval)
		autoIndexer.Start()

		// Wait for interrupt
		select {}

	case "help":
		printHelp()

	default:
		fmt.Printf("Unknown command: %s\n\n", *command)
		printHelp()
		os.Exit(1)
	}
}

func printSearchResults(results []*embedding.SearchResult) {
	if len(results) == 0 {
		fmt.Println("No results found")
		return
	}

	fmt.Printf("Found %d results:\n\n", len(results))
	for i, result := range results {
		if result.Tweet == nil {
			continue
		}

		fmt.Printf("%d. Score: %.3f | Distance: %.3f\n", i+1, result.Score, result.Distance)
		fmt.Printf("   User: @%s (%s)\n", result.Tweet.ScreenName, result.Tweet.UserName)
		fmt.Printf("   Tweet ID: %d\n", result.Tweet.TweetID)
		fmt.Printf("   Time: %s\n", result.Tweet.TweetTime.Format("2006-01-02 15:04:05"))
		fmt.Printf("   Content: %s\n", truncateText(result.Tweet.Content, 100))

		// Print metadata if available
		if result.Tweet.Metadata != nil {
			if isWeb3, ok := result.Tweet.Metadata["is_web3"].(bool); ok && isWeb3 {
				fmt.Printf("   Web3: ✓")
				if tokens, ok := result.Tweet.Metadata["web3_tokens"].([]string); ok {
					fmt.Printf(" (tokens: %v)", tokens)
				}
				fmt.Printf("\n")
			}
			if sentiment, ok := result.Tweet.Metadata["sentiment"].(string); ok {
				fmt.Printf("   Sentiment: %s", sentiment)
				if score, ok := result.Tweet.Metadata["sentiment_score"].(float32); ok {
					fmt.Printf(" (%.2f)", score)
				}
				fmt.Printf("\n")
			}
		}
		fmt.Println()
	}
}

func printStats(stats *embedding.IndexStats) {
	fmt.Println("Embedding Index Statistics:")
	fmt.Println("===========================")
	fmt.Printf("Total Tweets: %d\n", stats.TotalTweets)
	fmt.Printf("Indexed Tweets: %d\n", stats.IndexedTweets)
	fmt.Printf("Failed Tweets: %d\n", stats.FailedTweets)
	fmt.Printf("Collection Size: %d\n", stats.CollectionSize)
	fmt.Printf("Embedding Model: %s\n", stats.EmbeddingModel)
	fmt.Printf("Last Indexed: %s\n", stats.LastIndexedAt.Format("2006-01-02 15:04:05"))
	if stats.AverageScore > 0 {
		fmt.Printf("Average Score: %.3f\n", stats.AverageScore)
	}
}

func printHelp() {
	fmt.Println("xSync Tweet Embedder")
	fmt.Println("===================")
	fmt.Println()
	fmt.Println("This tool manages tweet embeddings in ChromaDB for semantic search.")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  index-all      Index all tweets from the database")
	fmt.Println("  index-new      Index new tweets (last 24 hours)")
	fmt.Println("  index-user     Index tweets for a specific user (requires -user)")
	fmt.Println("  search         Search tweets semantically (requires -query)")
	fmt.Println("  search-web3    Search web3-related tweets (requires -query)")
	fmt.Println("  stats          Show indexing statistics")
	fmt.Println("  auto           Start auto-indexer daemon")
	fmt.Println("  help           Show this help message")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -db string          SQLite database path (default: ./conf/data/xSync.db)")
	fmt.Println("  -chroma-url string  ChromaDB URL (default: http://localhost:8000)")
	fmt.Println("  -chroma-token string ChromaDB token (default: xsync-dev-token-2025)")
	fmt.Println("  -redis-url string   Redis URL (default: localhost:6379)")
	fmt.Println("  -redis-pass string  Redis password (default: xsync-redis-2025)")
	fmt.Println("  -user string        User ID for user-specific operations")
	fmt.Println("  -query string       Search query")
	fmt.Println("  -limit int          Search result limit (default: 50)")
	fmt.Println("  -interval duration  Auto-indexing interval (default: 5m)")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  # Index all tweets")
	fmt.Println("  ./embedder -cmd=index-all")
	fmt.Println()
	fmt.Println("  # Search for DeFi-related tweets")
	fmt.Println("  ./embedder -cmd=search-web3 -query=\"DeFi yield farming\"")
	fmt.Println()
	fmt.Println("  # Index tweets for a specific user")
	fmt.Println("  ./embedder -cmd=index-user -user=123456789")
	fmt.Println()
	fmt.Println("  # Start auto-indexer")
	fmt.Println("  ./embedder -cmd=auto -interval=10m")
}

func truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen-3] + "..."
}
