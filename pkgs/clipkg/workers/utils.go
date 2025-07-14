package workers

import (
	"context"
	"time"
)

// DrainChannel safely drains all remaining items from a channel
// This is useful for collecting any remaining work items after producers/consumers finish
func DrainChannel[T any](ch <-chan T) []T {
	var items []T
	for {
		select {
		case item, ok := <-ch:
			if !ok {
				return items
			}
			items = append(items, item)
		default:
			return items
		}
	}
}

// SafeChannelSend attempts to send to a channel with timeout and context cancellation
// Returns true if the send was successful, false if it timed out or was cancelled
func SafeChannelSend[T any](ctx context.Context, ch chan<- T, item T, timeout time.Duration) bool {
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case ch <- item:
		return true
	case <-ctx.Done():
		return false
	case <-timer.C:
		return false
	}
}

// BatchProcessor processes items in batches using the provided processor function
// This is useful for processing items in chunks to improve efficiency
func BatchProcessor[T any, R any](
	items []T,
	batchSize int,
	processor func(batch []T) []R,
) []R {
	var allResults []R

	for i := 0; i < len(items); i += batchSize {
		end := i + batchSize
		if end > len(items) {
			end = len(items)
		}

		batch := items[i:end]
		results := processor(batch)
		allResults = append(allResults, results...)
	}

	return allResults
}

// RetryWithBackoff retries a function with exponential backoff
// This is useful for handling transient failures with increasing delays
func RetryWithBackoff[T any](
	ctx context.Context,
	fn func() (T, error),
	maxRetries int,
	initialDelay time.Duration,
) (T, error) {
	var zero T
	delay := initialDelay

	for i := 0; i < maxRetries; i++ {
		result, err := fn()
		if err == nil {
			return result, nil
		}

		if i == maxRetries-1 {
			return zero, err
		}

		select {
		case <-ctx.Done():
			return zero, ctx.Err()
		case <-time.After(delay):
			delay *= 2 // exponential backoff
		}
	}

	return zero, context.DeadlineExceeded
}
