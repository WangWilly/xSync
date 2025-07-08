package workers

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
)

// ExampleTweetMediaDownload shows how to refactor the original tweet media download worker
// to use the SimpleWorker utility. This is a simplified example for demonstration.

// TweetWorkItem represents a tweet that needs media downloaded
type TweetWorkItem struct {
	TweetID   string
	MediaURLs []string
	UserID    string
}

// FailedDownload represents a failed download attempt
type FailedDownload struct {
	TweetID string
	Reason  string
}

// TweetMediaDownloader shows how to use SimpleWorker for tweet media downloads
type TweetMediaDownloader struct {
	worker     *SimpleWorker[TweetWorkItem, FailedDownload]
	maxWorkers int
}

// NewTweetMediaDownloader creates a new tweet media downloader
func NewTweetMediaDownloader(ctx context.Context, cancel context.CancelFunc, maxWorkers int) *TweetMediaDownloader {
	return &TweetMediaDownloader{
		worker:     NewSimpleWorker[TweetWorkItem, FailedDownload](ctx, cancel, maxWorkers),
		maxWorkers: maxWorkers,
	}
}

// DownloadTweetMedia downloads media for tweets using the SimpleWorker pattern
func (tmd *TweetMediaDownloader) DownloadTweetMedia(tweets []TweetWorkItem) ([]FailedDownload, error) {
	// Producer function that feeds tweets to the work channel
	producer := func(ctx context.Context, output chan<- TweetWorkItem) ([]TweetWorkItem, error) {
		var unsent []TweetWorkItem
		for _, tweet := range tweets {
			select {
			case <-ctx.Done():
				// Add remaining tweets to unsent list
				unsent = append(unsent, tweet)
				return unsent, ctx.Err()
			case output <- tweet:
				tmd.worker.IncrementProduced()
				log.WithField("tweetID", tweet.TweetID).Debug("Tweet queued for download")
			}
		}
		return unsent, nil
	}

	// Consumer function that downloads media for each tweet
	consumer := func(ctx context.Context, input <-chan TweetWorkItem) []FailedDownload {
		var failed []FailedDownload

		for {
			select {
			case <-ctx.Done():
				return failed
			case tweet, ok := <-input:
				if !ok {
					return failed
				}

				tmd.worker.IncrementConsumed()

				// Simulate media download
				if err := tmd.downloadMediaForTweet(ctx, tweet); err != nil {
					failed = append(failed, FailedDownload{
						TweetID: tweet.TweetID,
						Reason:  err.Error(),
					})
					log.WithFields(log.Fields{
						"tweetID": tweet.TweetID,
						"error":   err.Error(),
					}).Warn("Failed to download tweet media")
				} else {
					log.WithField("tweetID", tweet.TweetID).Debug("Tweet media downloaded successfully")
				}
			}
		}
	}

	// Run the producer-consumer pipeline
	result := tmd.worker.Process(producer, consumer, tmd.maxWorkers*2)

	// Log results
	log.WithFields(log.Fields{
		"produced": result.Stats.Produced,
		"consumed": result.Stats.Consumed,
		"failed":   result.Stats.Failed,
		"duration": result.Stats.Duration,
	}).Info("Tweet media download completed")

	return result.Failed, result.Error
}

// downloadMediaForTweet simulates downloading media for a single tweet
func (tmd *TweetMediaDownloader) downloadMediaForTweet(ctx context.Context, tweet TweetWorkItem) error {
	// This would be replaced with actual download logic
	for i, mediaURL := range tweet.MediaURLs {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Simulate download logic
			if err := tmd.downloadSingleMedia(ctx, mediaURL, tweet.TweetID, i); err != nil {
				return fmt.Errorf("failed to download media %d for tweet %s: %w", i, tweet.TweetID, err)
			}
		}
	}
	return nil
}

// downloadSingleMedia simulates downloading a single media file
func (tmd *TweetMediaDownloader) downloadSingleMedia(ctx context.Context, mediaURL, tweetID string, index int) error {
	// This would contain the actual HTTP download logic
	// For now, just simulate some work
	log.WithFields(log.Fields{
		"mediaURL": mediaURL,
		"tweetID":  tweetID,
		"index":    index,
	}).Debug("Downloading media file")

	// Simulate potential failure (10% failure rate for demo)
	// In real implementation, this would be actual download logic
	return nil
}

// ExampleUsage shows how to use the TweetMediaDownloader
func ExampleTweetMediaDownloaderUsage() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create downloader with 5 workers
	downloader := NewTweetMediaDownloader(ctx, cancel, 5)

	// Sample tweets to download
	tweets := []TweetWorkItem{
		{
			TweetID:   "tweet1",
			MediaURLs: []string{"https://example.com/media1.jpg", "https://example.com/media2.mp4"},
			UserID:    "user1",
		},
		{
			TweetID:   "tweet2",
			MediaURLs: []string{"https://example.com/media3.jpg"},
			UserID:    "user2",
		},
		// More tweets...
	}

	// Download media for all tweets
	failed, err := downloader.DownloadTweetMedia(tweets)

	if err != nil {
		log.WithError(err).Error("Producer error during download")
	}

	for _, failure := range failed {
		log.WithFields(log.Fields{
			"tweetID": failure.TweetID,
			"reason":  failure.Reason,
		}).Warn("Failed to download tweet media")
	}
}

// MigrationBenefits demonstrates the benefits of using SimpleWorker:
//
// 1. **Deadlock-resistant**: No condition variables, uses channels and atomics
// 2. **Clear separation**: Producer and consumer logic are separate functions
// 3. **Error handling**: Failed items are returned, not sent through error channels
// 4. **Statistics**: Built-in tracking of produced/consumed/failed counts
// 5. **Generic**: Can be reused for any producer-consumer pattern
// 6. **Testable**: Each component can be tested separately
// 7. **Maintainable**: Clear ownership and responsibility boundaries
//
// The original worker had potential deadlock issues with condition variables
// and complex error channel management. This refactored version using SimpleWorker
// eliminates those issues while providing better monitoring and error handling.
