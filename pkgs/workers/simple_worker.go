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
type SimpleWorker[T any] struct {
	ctx        context.Context
	cancel     context.CancelCauseFunc
	maxWorkers int

	// Counters for tracking
	produced int64
	consumed int64
}

// NewSimpleWorker creates a new simple worker with the specified number of workers
func NewSimpleWorker[T any](ctx context.Context, cancel context.CancelCauseFunc, maxWorkers int) *SimpleWorker[T] {
	return &SimpleWorker[T]{
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
type ProducerFunc[T any] func(ctx context.Context, cancel context.CancelCauseFunc, output chan<- T) ([]T, error)

// ConsumerFunc defines how to process work items
// The consumer should read from the input channel until it's closed
// and return any failed/unprocessed items
type ConsumerFunc[T any] func(ctx context.Context, cancel context.CancelCauseFunc, input <-chan T) []T

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
func (sw *SimpleWorker[T]) Process(
	producer ProducerFunc[T],
	consumer ConsumerFunc[T],
	bufferSize int,
) ProcessResult[T] {
	logger := log.WithField("function", "SimpleWorker.Process").
		WithField("maxWorkers", sw.maxWorkers).
		WithField("bufferSize", bufferSize)

	startTime := time.Now()

	var allFailed []T
	var failedMutex sync.Mutex

	workChan := make(chan T, bufferSize)
	var producerErr error
	go func() {
		defer close(workChan)

		logger := logger.WithField("worker", "producer")

		unsents, err := producer(sw.ctx, sw.cancel, workChan)
		logger.WithField("unsentCount", len(unsents)).
			WithField("error", err).
			Debug("producer finished")

		if err != nil {
			producerErr = err
			logger.WithError(err).Error("producer encountered an error")
			return
		}

		failedMutex.Lock()
		allFailed = append(allFailed, unsents...)
		failedMutex.Unlock()
	}()

	var wg sync.WaitGroup
	for i := 0; i < sw.maxWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			logger := logger.WithField("workerID", workerID)
			failed := consumer(sw.ctx, sw.cancel, workChan)
			logger.WithField("failedCount", len(failed)).Debug("consumer finished")

			failedMutex.Lock()
			allFailed = append(allFailed, failed...)
			failedMutex.Unlock()
		}(i)
	}
	wg.Wait()

	return ProcessResult[T]{
		Failed: allFailed,
		Error:  producerErr,
		Stats: ProcessStats{
			Produced: atomic.LoadInt64(&sw.produced),
			Consumed: atomic.LoadInt64(&sw.consumed),
			Failed:   int64(len(allFailed)),
			Duration: time.Since(startTime),
		},
	}
}

// IncrementProduced atomically increments the produced counter
// This should be called by the producer function for each item produced
func (sw *SimpleWorker[T]) IncrementProduced() {
	atomic.AddInt64(&sw.produced, 1)
}

// IncrementConsumed atomically increments the consumed counter
// This should be called by the consumer function for each item consumed
func (sw *SimpleWorker[T]) IncrementConsumed() {
	atomic.AddInt64(&sw.consumed, 1)
}

// GetStats returns current processing statistics
func (sw *SimpleWorker[T]) GetStats() ProcessStats {
	return ProcessStats{
		Produced: atomic.LoadInt64(&sw.produced),
		Consumed: atomic.LoadInt64(&sw.consumed),
	}
}
