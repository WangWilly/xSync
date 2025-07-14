package embedding

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/WangWilly/xSync/pkgs/model"
)

// EmbedderService handles text embedding generation
type EmbedderService struct {
	config     *Config
	httpClient *http.Client
}

// NewEmbedderService creates a new embedder service
func NewEmbedderService(config *Config) *EmbedderService {
	return &EmbedderService{
		config: config,
		httpClient: &http.Client{
			Timeout: config.RequestTimeout,
		},
	}
}

// EmbeddingRequest represents a request to generate embeddings
type EmbeddingRequest struct {
	Text  string `json:"text"`
	Model string `json:"model"`
}

// EmbeddingResponse represents the response from embedding service
type EmbeddingResponse struct {
	Embedding []float32 `json:"embedding"`
	Model     string    `json:"model"`
	Usage     struct {
		Tokens int `json:"tokens"`
	} `json:"usage"`
}

// GenerateEmbedding generates embedding for a single text
func (e *EmbedderService) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	// Try to call the Python embedding API first
	if embedding, err := e.callEmbeddingAPI(ctx, text); err == nil {
		return embedding, nil
	} else {
		// Log the error but continue with mock embedding for development
		fmt.Printf("Warning: Embedding API call failed: %v, using mock embedding\n", err)
	}

	// Fallback to mock embedding for development/testing
	return e.mockEmbedding(text), nil
}

// GenerateEmbeddings generates embeddings for multiple texts
func (e *EmbedderService) GenerateEmbeddings(ctx context.Context, texts []string) ([][]float32, error) {
	embeddings := make([][]float32, len(texts))
	for i, text := range texts {
		embedding, err := e.GenerateEmbedding(ctx, text)
		if err != nil {
			return nil, fmt.Errorf("failed to generate embedding for text %d: %w", i, err)
		}
		embeddings[i] = embedding
	}
	return embeddings, nil
}

// mockEmbedding generates a mock embedding (replace with real implementation)
func (e *EmbedderService) mockEmbedding(text string) []float32 {
	// This is a mock implementation - in production, use sentence-transformers
	embedding := make([]float32, e.config.EmbeddingDim)

	// Simple hash-based mock embedding
	hash := uint32(0)
	for _, char := range text {
		hash = hash*31 + uint32(char)
	}

	for i := range embedding {
		hash = hash*1103515245 + 12345
		embedding[i] = float32(int32(hash)%2000-1000) / 1000.0
	}

	// Normalize the embedding
	var norm float32
	for _, val := range embedding {
		norm += val * val
	}
	norm = float32(1.0 / (float64(norm) + 1e-8))

	for i := range embedding {
		embedding[i] *= norm
	}

	return embedding
}

// TweetProcessor processes tweets for embedding
type TweetProcessor struct {
	embedder *EmbedderService
	config   *Config
}

// NewTweetProcessor creates a new tweet processor
func NewTweetProcessor(config *Config) *TweetProcessor {
	return &TweetProcessor{
		embedder: NewEmbedderService(config),
		config:   config,
	}
}

// ProcessTweet converts a model.Tweet to TweetEmbedding
func (p *TweetProcessor) ProcessTweet(tweet *model.Tweet, user *model.User) (*TweetEmbedding, error) {
	ctx := context.Background()

	// Generate embedding for tweet content
	embedding, err := p.embedder.GenerateEmbedding(ctx, tweet.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	// Create tweet embedding
	tweetEmbedding := &TweetEmbedding{
		ID:        fmt.Sprintf("tweet_%d", tweet.TweetId),
		TweetID:   tweet.TweetId,
		UserID:    tweet.UserId,
		Content:   tweet.Content,
		TweetTime: tweet.TweetTime,
		Embedding: embedding,
		CreatedAt: time.Now(),
	}

	// Add user information if available
	if user != nil {
		tweetEmbedding.UserName = user.Name
		tweetEmbedding.ScreenName = user.ScreenName
	}

	// Analyze for web3 content
	tweetEmbedding.Metadata = p.analyzeWeb3Content(tweet.Content)

	return tweetEmbedding, nil
}

// ProcessTweets processes multiple tweets
func (p *TweetProcessor) ProcessTweets(tweets []*model.Tweet, users map[uint64]*model.User) ([]*TweetEmbedding, error) {
	embeddings := make([]*TweetEmbedding, 0, len(tweets))

	for _, tweet := range tweets {
		user := users[tweet.UserId]
		embedding, err := p.ProcessTweet(tweet, user)
		if err != nil {
			// Log error but continue processing other tweets
			fmt.Printf("Error processing tweet %d: %v\n", tweet.TweetId, err)
			continue
		}
		embeddings = append(embeddings, embedding)
	}

	return embeddings, nil
}

// analyzeWeb3Content analyzes tweet content for web3-related terms
func (p *TweetProcessor) analyzeWeb3Content(content string) map[string]interface{} {
	metadata := make(map[string]interface{})

	// Web3 keywords to detect
	web3Keywords := []string{
		"bitcoin", "btc", "ethereum", "eth", "defi", "nft", "dao", "dex",
		"yield", "farming", "staking", "liquidity", "airdrop", "token",
		"crypto", "blockchain", "smart contract", "metamask", "uniswap",
		"opensea", "polygon", "solana", "binance", "coinbase", "web3",
		"metaverse", "gamefi", "play to earn", "p2e", "hodl", "moon",
		"pump", "dump", "bullish", "bearish", "altcoin", "memecoin",
	}

	contentLower := string(bytes.ToLower([]byte(content)))
	var detectedTokens []string

	for _, keyword := range web3Keywords {
		if bytes.Contains([]byte(contentLower), []byte(keyword)) {
			detectedTokens = append(detectedTokens, keyword)
		}
	}

	if len(detectedTokens) > 0 {
		metadata["web3_tokens"] = detectedTokens
		metadata["is_web3"] = true
		metadata["web3_score"] = float32(len(detectedTokens)) / 10.0 // Simple scoring
	} else {
		metadata["is_web3"] = false
		metadata["web3_score"] = 0.0
	}

	// Sentiment analysis (simple keyword-based)
	positive := []string{"good", "great", "amazing", "bullish", "moon", "pump", "buy"}
	negative := []string{"bad", "terrible", "bearish", "dump", "crash", "sell", "scam"}

	var sentiment string
	var sentimentScore float32

	positiveCount := 0
	negativeCount := 0

	for _, word := range positive {
		if bytes.Contains([]byte(contentLower), []byte(word)) {
			positiveCount++
		}
	}

	for _, word := range negative {
		if bytes.Contains([]byte(contentLower), []byte(word)) {
			negativeCount++
		}
	}

	if positiveCount > negativeCount {
		sentiment = "positive"
		sentimentScore = float32(positiveCount) / float32(positiveCount+negativeCount)
	} else if negativeCount > positiveCount {
		sentiment = "negative"
		sentimentScore = -float32(negativeCount) / float32(positiveCount+negativeCount)
	} else {
		sentiment = "neutral"
		sentimentScore = 0.0
	}

	metadata["sentiment"] = sentiment
	metadata["sentiment_score"] = sentimentScore

	return metadata
}

// callEmbeddingAPI calls the Python embedding API
func (e *EmbedderService) callEmbeddingAPI(ctx context.Context, text string) ([]float32, error) {
	request := EmbeddingRequest{
		Text:  text,
		Model: e.config.EmbeddingModel,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Call the embedding API service
	apiURL := "http://localhost:8001/embed"
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("embedding API returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var response EmbeddingResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return response.Embedding, nil
}
