package workers

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	log "github.com/sirupsen/logrus"
)

// SimpleWorker provides a clean, minimal producer-consumer pattern
// that's generic and reusable for any type of work items.
//
// Type parameters:
// - T: The type of work items produced and consumed
// - R: The type of results/failed items returned by consumers
type SimpleWorker[T any, R any] struct {
	ctx        context.Context
	cancel     context.CancelFunc
	maxWorkers int

	// Counters for tracking
	produced int64
	consumed int64
}

// NewSimpleWorker creates a new simple worker with the specified number of workers
func NewSimpleWorker[T any, R any](ctx context.Context, cancel context.CancelFunc, maxWorkers int) *SimpleWorker[T, R] {
	return &SimpleWorker[T, R]{
		ctx:        ctx,
		cancel:     cancel,
		maxWorkers: maxWorkers,
		produced:   0,
		consumed:   0,
	}
}

// ProducerFunc defines how to generate work items
// The producer should send items to the output channel and close it when done
// Returns an error and a slice of items that couldn't be sent
type ProducerFunc[T any] func(ctx context.Context, output chan<- T) ([]T, error)

// ConsumerFunc defines how to process work items
// The consumer should read from the input channel until it's closed
// and return any failed/unprocessed items
type ConsumerFunc[T any, R any] func(ctx context.Context, input <-chan T) []R

// ProcessResult contains the results of processing
type ProcessResult[R any] struct {
	Failed []R
	Error  error
	Stats  ProcessStats
}

// ProcessStats contains processing statistics
type ProcessStats struct {
	Produced int64
	Consumed int64
	Failed   int64
	Duration time.Duration
}

// Process runs the producer-consumer pipeline
//
// Parameters:
// - producer: Function that generates work items
// - consumer: Function that processes work items and returns failed items
// - bufferSize: Size of the work channel buffer
//
// Returns:
// - ProcessResult containing failed items, errors, and statistics
func (sw *SimpleWorker[T, R]) Process(
	producer ProducerFunc[T],
	consumer ConsumerFunc[T, R],
	bufferSize int,
) ProcessResult[R] {
	startTime := time.Now()

	// Create work channel
	workChan := make(chan T, bufferSize)

	// Producer error and unsent items channels
	producerErr := make(chan error, 1)
	producerUnsent := make(chan []T, 1)

	// Start producer
	go func() {
		defer close(workChan)
		unsent, err := producer(sw.ctx, workChan)
		if err != nil {
			select {
			case producerErr <- err:
			default:
			}
		}
		if len(unsent) > 0 {
			select {
			case producerUnsent <- unsent:
			default:
			}
		}
	}()

	// Start consumers
	var wg sync.WaitGroup
	var allFailed [][]R
	var failedMutex sync.Mutex

	for i := 0; i < sw.maxWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			logger := log.WithField("workerID", workerID)
			logger.Debug("consumer started")

			failed := consumer(sw.ctx, workChan)

			failedMutex.Lock()
			allFailed = append(allFailed, failed)
			failedMutex.Unlock()

			logger.WithField("failedCount", len(failed)).Debug("consumer finished")
		}(i)
	}

	// Wait for all consumers to finish
	wg.Wait()

	// Collect all failed items
	var totalFailed []R
	for _, consumerFailed := range allFailed {
		totalFailed = append(totalFailed, consumerFailed...)
	}

	// Add unsent items from producer to failed list
	select {
	case unsent := <-producerUnsent:
		for _, item := range unsent {
			// Convert T to R - assuming they are the same type for tweet processing
			if convertedItem, ok := any(item).(R); ok {
				totalFailed = append(totalFailed, convertedItem)
			}
		}
	default:
	}

	// Check for producer errors
	var err error
	select {
	case err = <-producerErr:
	default:
	}

	duration := time.Since(startTime)

	return ProcessResult[R]{
		Failed: totalFailed,
		Error:  err,
		Stats: ProcessStats{
			Produced: atomic.LoadInt64(&sw.produced),
			Consumed: atomic.LoadInt64(&sw.consumed),
			Failed:   int64(len(totalFailed)),
			Duration: duration,
		},
	}
}

// IncrementProduced atomically increments the produced counter
// This should be called by the producer function for each item produced
func (sw *SimpleWorker[T, R]) IncrementProduced() {
	atomic.AddInt64(&sw.produced, 1)
}

// IncrementConsumed atomically increments the consumed counter
// This should be called by the consumer function for each item consumed
func (sw *SimpleWorker[T, R]) IncrementConsumed() {
	atomic.AddInt64(&sw.consumed, 1)
}

// GetStats returns current processing statistics
func (sw *SimpleWorker[T, R]) GetStats() ProcessStats {
	return ProcessStats{
		Produced: atomic.LoadInt64(&sw.produced),
		Consumed: atomic.LoadInt64(&sw.consumed),
	}
}
