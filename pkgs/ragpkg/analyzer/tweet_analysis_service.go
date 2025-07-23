package analyzer

import (
	"context"
	"time"

	"github.com/WangWilly/xSync/pkgs/commonpkg/model"
	log "github.com/sirupsen/logrus"
)

// TweetAnalysisService provides integration with existing tweet processing
type TweetAnalysisService struct {
	ragAnalyzer *RAGAnalyzer
	enabled     bool
	logger      *log.Entry
}

// NewTweetAnalysisService creates a new tweet analysis service
func NewTweetAnalysisService(ragAnalyzer *RAGAnalyzer, enabled bool) *TweetAnalysisService {
	return &TweetAnalysisService{
		ragAnalyzer: ragAnalyzer,
		enabled:     enabled,
		logger:      log.WithField("service", "tweet_analysis"),
	}
}

// AnalyzeNewTweet analyzes a newly saved tweet for token mentions
func (t *TweetAnalysisService) AnalyzeNewTweet(ctx context.Context, tweet *model.Tweet) {
	if !t.enabled {
		return
	}

	go func() {
		// Run analysis in background to avoid blocking tweet saving
		defer func() {
			if r := recover(); r != nil {
				t.logger.WithError(r.(error)).Error("Panic during tweet analysis")
			}
		}()

		// Add small delay to ensure tweet is properly saved
		time.Sleep(100 * time.Millisecond)

		result, err := t.ragAnalyzer.AnalyzeTweet(ctx, tweet)
		if err != nil {
			t.logger.WithError(err).WithField("tweet_id", tweet.TweetId).Error("Failed to analyze tweet")
			return
		}

		if len(result.RecommendedTokens) > 0 {
			t.logger.WithFields(log.Fields{
				"tweet_id":           tweet.TweetId,
				"user_id":            tweet.UserId,
				"recommended_tokens": result.RecommendedTokens,
				"confidence_score":   result.ConfidenceScore,
				"content_preview":    truncateString(tweet.Content, 100),
			}).Info("Found potential token mentions in new tweet")

			// Here you could add additional processing like:
			// - Storing analysis results in a separate table
			// - Sending notifications for high-confidence matches
			// - Triggering LLM analysis for further validation
			// - Adding to a queue for manual review
		}
	}()
}

// truncateString truncates a string to maxLength characters
func truncateString(s string, maxLength int) string {
	if len(s) <= maxLength {
		return s
	}
	return s[:maxLength] + "..."
}
