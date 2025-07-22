package main

import (
	"context"
	"log"
	"time"

	"github.com/WangWilly/xSync/pkgs/commonpkg/clients/chromatokenclient"
	"github.com/WangWilly/xSync/pkgs/commonpkg/model"
	"github.com/WangWilly/xSync/pkgs/commonpkg/repos/tokenrepo"
	"github.com/WangWilly/xSync/pkgs/commonpkg/repos/tweetrepo"
	"github.com/WangWilly/xSync/pkgs/ragpkg/analyzer"
	"github.com/jmoiron/sqlx"
)

// Example showing how to integrate the RAG analyzer into existing tweet processing
func main() {
	// Note: This is just an example - you'll need proper database connection
	// and ChromaDB setup in a real implementation
	var db *sqlx.DB // This should be properly initialized

	// Initialize ChromaDB client
	chromaClient, err := chromatokenclient.New("http://localhost:8000") // Adjust URL as needed
	if err != nil {
		log.Fatal("Failed to create ChromaDB client:", err)
	}

	// Initialize repositories
	tweetRepo := tweetrepo.New()
	tokenRepo := tokenrepo.New()

	// Create RAG analyzer
	ragAnalyzer := analyzer.NewRAGAnalyzer(db, chromaClient, tweetRepo, tokenRepo)

	// Create tweet analysis service with async enabled
	tweetAnalysisService := analyzer.NewTweetAnalysisService(ragAnalyzer, true)

	// Example tweet that might contain token mentions
	tweet := &model.Tweet{
		Id:        1,
		TweetId:   1234567890,
		UserId:    9876543210,
		Content:   "Just bought some $BTC and looking at $ETH trends. What do you think about #DeFi and $LINK potential?",
		TweetTime: time.Now(),
	}

	// Analyze the tweet for token mentions
	ctx := context.Background()
	tweetAnalysisService.AnalyzeNewTweet(ctx, tweet)

	// Example of direct analysis
	result, err := ragAnalyzer.AnalyzeTweet(ctx, tweet)
	if err != nil {
		log.Printf("Analysis failed: %v", err)
		return
	}

	log.Printf("Analysis completed for tweet: %s", tweet.Content)
	log.Printf("Found %d potential tokens", len(result.PotentialTokens))
	for _, token := range result.PotentialTokens {
		log.Printf("- Potential token: %s", token)
	}

	log.Printf("Found %d ChromaDB matches", len(result.ChromaMatches))
	for _, token := range result.ChromaMatches {
		log.Printf("- ChromaDB match: %s", token)
	}

	log.Printf("Found %d recommended tokens", len(result.RecommendedTokens))
	for _, token := range result.RecommendedTokens {
		log.Printf("- Recommended token: %s", token)
	}
}
