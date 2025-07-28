package analyzer

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/WangWilly/xSync/pkgs/commonpkg/model"
	chroma "github.com/amikos-tech/chroma-go/pkg/api/v2"
	"github.com/jmoiron/sqlx"
	log "github.com/sirupsen/logrus"
)

// TweetRepo interface for tweet operations
type TweetRepo interface {
	GetById(ctx context.Context, db *sqlx.DB, id int64) (*model.Tweet, error)
	ListByUserId(ctx context.Context, db *sqlx.DB, userId uint64) ([]*model.Tweet, error)
	GetByTweetId(ctx context.Context, db *sqlx.DB, tweetId uint64) (*model.Tweet, error)
	Update(ctx context.Context, db *sqlx.DB, tweet *model.Tweet) error
	Create(ctx context.Context, db *sqlx.DB, tweet *model.Tweet) error
}

// TokenRepo interface for token operations
type TokenRepo interface {
	SearchTokens(ctx context.Context, db *sqlx.DB, searchTerm string, limit int) ([]model.Token, error)
}

// ChromaTokenClient interface for ChromaDB operations
type ChromaTokenClient interface {
	GetTokens(ctx context.Context, query string, limit int) (chroma.QueryResult, error)
}

// RAGAnalyzer provides RAG functionality for tweet analysis
type RAGAnalyzer struct {
	db                *sqlx.DB
	chromaClient      ChromaTokenClient
	tweetRepo         TweetRepo
	tokenRepo         TokenRepo
	logger            *log.Entry
	lastProcessedTime time.Time
}

// NewRAGAnalyzer creates a new RAG analyzer
func NewRAGAnalyzer(
	db *sqlx.DB,
	chromaClient ChromaTokenClient,
	tweetRepo TweetRepo,
	tokenRepo TokenRepo,
) *RAGAnalyzer {
	return &RAGAnalyzer{
		db:                db,
		chromaClient:      chromaClient,
		tweetRepo:         tweetRepo,
		tokenRepo:         tokenRepo,
		logger:            log.WithField("service", "rag_analyzer"),
		lastProcessedTime: time.Now().Add(-24 * time.Hour), // Start from 24 hours ago
	}
}

// TweetAnalysisResult represents the result of analyzing a tweet for token mentions
type TweetAnalysisResult struct {
	TweetID           uint64   `json:"tweet_id"`
	Content           string   `json:"content"`
	PotentialTokens   []string `json:"potential_tokens"`
	ChromaMatches     []string `json:"chroma_matches"`
	ConfidenceScore   float64  `json:"confidence_score"`
	RecommendedTokens []string `json:"recommended_tokens"`
}

// StartContinuousAnalysis starts the continuous analysis process
func (r *RAGAnalyzer) StartContinuousAnalysis(ctx context.Context) error {
	r.logger.Info("Starting continuous tweet analysis for token detection")

	ticker := time.NewTicker(30 * time.Second) // Check every 30 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			r.logger.Info("Stopping continuous analysis due to context cancellation")
			return ctx.Err()
		case <-ticker.C:
			if err := r.processNewTweets(ctx); err != nil {
				r.logger.WithError(err).Error("Error processing new tweets")
			}
		}
	}
}

// processNewTweets processes tweets that haven't been analyzed yet
func (r *RAGAnalyzer) processNewTweets(ctx context.Context) error {
	// Query for tweets created after the last processed time
	query := `
		SELECT id, user_id, tweet_id, content, tweet_time, created_at, updated_at
		FROM tweets 
		WHERE created_at > $1 
		ORDER BY created_at ASC
		LIMIT 100
	`

	var tweets []model.Tweet
	err := r.db.SelectContext(ctx, &tweets, query, r.lastProcessedTime)
	if err != nil {
		return fmt.Errorf("failed to get new tweets: %w", err)
	}

	if len(tweets) == 0 {
		return nil // No new tweets to process
	}

	r.logger.Infof("Processing %d new tweets for token analysis", len(tweets))

	for _, tweet := range tweets {
		result, err := r.AnalyzeTweet(ctx, &tweet)
		if err != nil {
			r.logger.WithError(err).WithField("tweet_id", tweet.TweetId).Error("Failed to analyze tweet")
			continue
		}

		if len(result.RecommendedTokens) > 0 {
			r.logger.WithFields(log.Fields{
				"tweet_id":           tweet.TweetId,
				"recommended_tokens": result.RecommendedTokens,
				"confidence_score":   result.ConfidenceScore,
			}).Info("Found potential token mentions in tweet")
		}

		// Update the last processed time
		if tweet.CreatedAt.After(r.lastProcessedTime) {
			r.lastProcessedTime = tweet.CreatedAt
		}
	}

	return nil
}

// AnalyzeTweet analyzes a single tweet for potential token symbols
func (r *RAGAnalyzer) AnalyzeTweet(ctx context.Context, tweet *model.Tweet) (*TweetAnalysisResult, error) {
	result := &TweetAnalysisResult{
		TweetID:         tweet.TweetId,
		Content:         tweet.Content,
		PotentialTokens: []string{},
		ChromaMatches:   []string{},
		ConfidenceScore: 0.0,
	}

	// Step 1: Extract potential token symbols using pattern matching
	potentialTokens := r.extractPotentialTokens(tweet.Content)
	result.PotentialTokens = potentialTokens

	// Step 2: Query ChromaDB for semantic matches
	chromaMatches, err := r.queryChromaForTokens(ctx, tweet.Content)
	if err != nil {
		r.logger.WithError(err).Warn("Failed to query ChromaDB for tokens")
	} else {
		result.ChromaMatches = chromaMatches
	}

	// Step 3: Combine and rank the results
	result.RecommendedTokens = r.combineAndRankTokens(potentialTokens, chromaMatches)
	result.ConfidenceScore = r.calculateConfidenceScore(tweet.Content, result.RecommendedTokens)

	return result, nil
}

// extractPotentialTokens uses regex and patterns to find potential token symbols
func (r *RAGAnalyzer) extractPotentialTokens(content string) []string {
	var tokens []string

	// Convert content to uppercase for case-insensitive matching
	upperContent := strings.ToUpper(content)

	// Common patterns for token mentions
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`\$([A-Z][A-Z0-9_]{2,15})\b`),             // $BTC, $ETH, $SOL (3+ chars)
		regexp.MustCompile(`\#([A-Z][A-Z0-9_]{2,15})\b`),             // #BTC, #Solana (3+ chars)
		regexp.MustCompile(`\b([A-Z]{3,6})\s+(?:TOKEN|COIN)\b`),      // BTC token, ETH coin
		regexp.MustCompile(`\b([A-Z]{3,6})\s+(?:TO|AT)\s+\$`),        // SOL to $, BTC at $
		regexp.MustCompile(`\b([A-Z]{3,6})\s+(?:PRICE|PUMP|MOON)\b`), // SOL price, BTC pump
	}

	for _, pattern := range patterns {
		matches := pattern.FindAllStringSubmatch(upperContent, -1)
		for _, match := range matches {
			if len(match) > 1 {
				token := match[1]
				// Clean up underscores and normalize
				token = strings.ReplaceAll(token, "_", "_")
				// Filter out common words that aren't tokens
				if !r.isCommonWord(token) && len(token) >= 3 && len(token) <= 15 {
					tokens = append(tokens, token)
				}
			}
		}
	}

	// Remove duplicates
	return r.removeDuplicates(tokens)
}

// queryChromaForTokens queries ChromaDB for semantic matches
func (r *RAGAnalyzer) queryChromaForTokens(ctx context.Context, content string) ([]string, error) {
	// Query ChromaDB with the tweet content
	results, err := r.chromaClient.GetTokens(ctx, content, 5)
	if err != nil {
		return nil, err
	}

	// Parse ChromaDB results and extract token symbols
	var tokens []string

	// Extract token symbols from ChromaDB results
	if results != nil && results.CountGroups() > 0 {
		metadataGroups := results.GetMetadatasGroups()
		for _, metadataGroup := range metadataGroups {
			for _, metadata := range metadataGroup {
				if metadata != nil {
					if symbol, exists := metadata.GetString("symbol"); exists {
						tokens = append(tokens, strings.ToUpper(symbol))
					}
				}
			}
		}
	}

	return r.removeDuplicates(tokens), nil
}

// combineAndRankTokens combines pattern matches and ChromaDB results
func (r *RAGAnalyzer) combineAndRankTokens(patternTokens, chromaTokens []string) []string {
	tokenScore := make(map[string]int)

	// Score pattern matches
	for _, token := range patternTokens {
		tokenScore[token] += 2 // Pattern matches get higher score
	}

	// Score ChromaDB matches
	for _, token := range chromaTokens {
		tokenScore[token] += 3 // Semantic matches get highest score
	}

	// Sort by score and return top tokens
	var rankedTokens []string
	for token, score := range tokenScore {
		if score >= 2 { // Minimum threshold
			rankedTokens = append(rankedTokens, token)
		}
	}

	return rankedTokens
}

// calculateConfidenceScore calculates confidence based on various factors
func (r *RAGAnalyzer) calculateConfidenceScore(content string, tokens []string) float64 {
	if len(tokens) == 0 {
		return 0.0
	}

	score := 0.0
	contentLower := strings.ToLower(content)

	// Factor 1: Number of token mentions
	score += float64(len(tokens)) * 0.2

	// Factor 2: Presence of financial keywords
	financialKeywords := []string{
		"price", "pump", "moon", "buy", "sell", "trade", "investing",
		"bullish", "bearish", "hodl", "dip", "ath", "support", "resistance",
	}

	for _, keyword := range financialKeywords {
		if strings.Contains(contentLower, keyword) {
			score += 0.1
		}
	}

	// Factor 3: Presence of currency symbols
	if strings.Contains(content, "$") {
		score += 0.2
	}

	// Normalize to 0-1 range
	if score > 1.0 {
		score = 1.0
	}

	return score
}

// isCommonWord filters out common English words that aren't tokens
func (r *RAGAnalyzer) isCommonWord(word string) bool {
	commonWords := map[string]bool{
		"THE": true, "AND": true, "FOR": true, "ARE": true, "BUT": true,
		"NOT": true, "YOU": true, "ALL": true, "CAN": true, "HER": true,
		"WAS": true, "ONE": true, "OUR": true, "HAD": true, "OUT": true,
		"DAY": true, "GET": true, "HAS": true, "HIM": true, "HOW": true,
		"NEW": true, "NOW": true, "OLD": true, "SEE": true, "TWO": true,
		"WAY": true, "WHO": true, "BOY": true, "DID": true, "LET": true,
		"PUT": true, "SAY": true, "SHE": true, "TOO": true, "USE": true,
		"WILL": true, "MOON": true, "LOVE": true, "THINK": true, "GREAT": true,
	}
	return commonWords[word]
}

// removeDuplicates removes duplicate tokens from a slice
func (r *RAGAnalyzer) removeDuplicates(tokens []string) []string {
	seen := make(map[string]bool)
	var unique []string

	for _, token := range tokens {
		if !seen[token] {
			seen[token] = true
			unique = append(unique, token)
		}
	}

	return unique
}
