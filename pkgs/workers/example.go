package workers

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	log "github.com/sirupsen/logrus"
)

// Example demonstrates how to use the SimpleWorker for different scenarios

// ExampleWorkItem represents a unit of work
type ExampleWorkItem struct {
	ID   int
	Data string
}

// ExampleResult represents the result of processing a work item
type ExampleResult struct {
	ID    int
	Error string
}

// ExampleBasicUsage shows basic producer-consumer usage
func ExampleBasicUsage() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create a simple worker with 3 consumer workers
	worker := NewSimpleWorker[ExampleWorkItem, ExampleResult](ctx, cancel, 3)

	// Define a producer that generates work items
	producer := func(ctx context.Context, output chan<- ExampleWorkItem) ([]ExampleWorkItem, error) {
		var unsent []ExampleWorkItem
		for i := 0; i < 10; i++ {
			select {
			case <-ctx.Done():
				// Add remaining items to unsent list
				for j := i; j < 10; j++ {
					unsent = append(unsent, ExampleWorkItem{ID: j, Data: fmt.Sprintf("item-%d", j)})
				}
				return unsent, ctx.Err()
			case output <- ExampleWorkItem{ID: i, Data: fmt.Sprintf("item-%d", i)}:
				worker.IncrementProduced()
			}
		}
		return unsent, nil
	}

	// Define a consumer that processes work items
	consumer := func(ctx context.Context, input <-chan ExampleWorkItem) []ExampleResult {
		var failed []ExampleResult

		for {
			select {
			case <-ctx.Done():
				return failed
			case item, ok := <-input:
				if !ok {
					return failed
				}

				worker.IncrementConsumed()

				// Simulate some processing
				time.Sleep(100 * time.Millisecond)

				// Simulate occasional failures
				if rand.Float32() < 0.1 { // 10% failure rate
					failed = append(failed, ExampleResult{
						ID:    item.ID,
						Error: "random failure",
					})
				}
			}
		}
	}

	// Run the producer-consumer pipeline
	result := worker.Process(producer, consumer, 5)

	// Handle results
	log.WithFields(log.Fields{
		"produced": result.Stats.Produced,
		"consumed": result.Stats.Consumed,
		"failed":   result.Stats.Failed,
		"duration": result.Stats.Duration,
	}).Info("Processing completed")

	if result.Error != nil {
		log.WithError(result.Error).Error("Producer error")
	}

	for _, failed := range result.Failed {
		log.WithFields(log.Fields{
			"id":    failed.ID,
			"error": failed.Error,
		}).Warn("Failed item")
	}
}

// ExampleWithRetry shows how to use RetryWithBackoff utility
func ExampleWithRetry() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Define a function that might fail
	unreliableOperation := func() (string, error) {
		if rand.Float32() < 0.7 { // 70% failure rate
			return "", fmt.Errorf("random failure")
		}
		return "success", nil
	}

	// Retry with exponential backoff
	result, err := RetryWithBackoff(
		ctx,
		unreliableOperation,
		5,                    // max retries
		100*time.Millisecond, // initial delay
	)

	if err != nil {
		log.WithError(err).Error("Operation failed after retries")
	} else {
		log.WithField("result", result).Info("Operation succeeded")
	}
}

// ExampleWithBatching shows how to use BatchProcessor utility
func ExampleWithBatching() {
	items := make([]ExampleWorkItem, 100)
	for i := 0; i < 100; i++ {
		items[i] = ExampleWorkItem{ID: i, Data: fmt.Sprintf("item-%d", i)}
	}

	// Process items in batches of 10
	results := BatchProcessor(
		items,
		10,
		func(batch []ExampleWorkItem) []ExampleResult {
			var failed []ExampleResult

			// Process batch
			for _, item := range batch {
				// Simulate occasional failures
				if rand.Float32() < 0.05 { // 5% failure rate
					failed = append(failed, ExampleResult{
						ID:    item.ID,
						Error: "batch processing failure",
					})
				}
			}

			return failed
		},
	)

	log.WithField("failedCount", len(results)).Info("Batch processing completed")
}

// ExampleWithChannelDraining shows how to use DrainChannel utility
func ExampleWithChannelDraining() {
	ch := make(chan ExampleWorkItem, 10)

	// Add some items to the channel
	for i := 0; i < 5; i++ {
		ch <- ExampleWorkItem{ID: i, Data: fmt.Sprintf("item-%d", i)}
	}

	// Drain all items from the channel
	items := DrainChannel(ch)

	log.WithField("drainedCount", len(items)).Info("Channel drained")
}

// ExampleWithSafeChannelSend shows how to use SafeChannelSend utility
func ExampleWithSafeChannelSend() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ch := make(chan ExampleWorkItem, 1)

	// Try to send with timeout
	success := SafeChannelSend(
		ctx,
		ch,
		ExampleWorkItem{ID: 1, Data: "test"},
		1*time.Second,
	)

	if success {
		log.Info("Send successful")
	} else {
		log.Warn("Send failed or timed out")
	}
}
