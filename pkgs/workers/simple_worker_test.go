package workers

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

func TestSimpleWorker_BasicUsage(t *testing.T) {
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)

	worker := NewSimpleWorker[int](ctx, cancel, 2)

	// Producer that generates 10 items
	producer := func(ctx context.Context, cancel context.CancelCauseFunc, output chan<- int) ([]int, error) {
		var unsent []int
		for i := 0; i < 10; i++ {
			select {
			case <-ctx.Done():
				// Add remaining items to unsent list
				for j := i; j < 10; j++ {
					unsent = append(unsent, j)
				}
				return unsent, ctx.Err()
			case output <- i:
				worker.IncrementProduced()
			}
		}
		return unsent, nil
	}

	// Consumer that processes items and fails odd numbers
	consumer := func(ctx context.Context, cancel context.CancelCauseFunc, input <-chan int) []int {
		var failed []int

		for {
			select {
			case <-ctx.Done():
				return failed
			case item, ok := <-input:
				if !ok {
					return failed
				}

				worker.IncrementConsumed()

				// Fail odd numbers
				if item%2 == 1 {
					failed = append(failed, item)
				}
			}
		}
	}

	result := worker.Process(producer, consumer, 5)

	// Verify results
	if result.Error != nil {
		t.Errorf("Expected no error, got %v", result.Error)
	}

	if result.Stats.Produced != 10 {
		t.Errorf("Expected 10 produced, got %d", result.Stats.Produced)
	}

	if result.Stats.Consumed != 10 {
		t.Errorf("Expected 10 consumed, got %d", result.Stats.Consumed)
	}

	if len(result.Failed) != 5 {
		t.Errorf("Expected 5 failed items, got %d", len(result.Failed))
	}

	// Verify all odd numbers are in failed list
	failedMap := make(map[int]bool)
	for _, failed := range result.Failed {
		failedMap[failed] = true
	}

	for i := 1; i < 10; i += 2 {
		if !failedMap[i] {
			t.Errorf("Expected failed item %d not found", i)
		}
	}
}

func TestSimpleWorker_ProducerError(t *testing.T) {
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)

	worker := NewSimpleWorker[int](ctx, cancel, 1)

	// Producer that returns an error
	producer := func(ctx context.Context, cancel context.CancelCauseFunc, output chan<- int) ([]int, error) {
		return nil, fmt.Errorf("producer error")
	}

	// Consumer that just consumes
	consumer := func(ctx context.Context, cancel context.CancelCauseFunc, input <-chan int) []int {
		var failed []int
		for item := range input {
			worker.IncrementConsumed()
			_ = item
		}
		return failed
	}

	result := worker.Process(producer, consumer, 5)

	if result.Error == nil {
		t.Error("Expected producer error, got nil")
	}

	if result.Error.Error() != "producer error" {
		t.Errorf("Expected 'producer error', got %v", result.Error)
	}
}

func TestSimpleWorker_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancelCause(context.Background())

	worker := NewSimpleWorker[int](ctx, cancel, 1)

	// Producer that generates items slowly
	producer := func(ctx context.Context, cancel context.CancelCauseFunc, output chan<- int) ([]int, error) {
		var unsent []int
		for i := 0; i < 100; i++ {
			select {
			case <-ctx.Done():
				// Add remaining items to unsent list
				for j := i; j < 100; j++ {
					unsent = append(unsent, j)
				}
				return unsent, ctx.Err()
			case output <- i:
				worker.IncrementProduced()
				time.Sleep(10 * time.Millisecond)
			}
		}
		return unsent, nil
	}

	// Consumer that processes items
	consumer := func(ctx context.Context, cancel context.CancelCauseFunc, input <-chan int) []int {
		var failed []int
		for {
			select {
			case <-ctx.Done():
				return failed
			case item, ok := <-input:
				if !ok {
					return failed
				}
				worker.IncrementConsumed()
				_ = item
			}
		}
	}

	// Cancel context after 50ms
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel(nil)
	}()

	result := worker.Process(producer, consumer, 5)

	// Should have some items produced but not all 100
	if result.Stats.Produced == 0 {
		t.Error("Expected some items produced")
	}

	if result.Stats.Produced >= 100 {
		t.Error("Expected production to be cancelled before completing")
	}
}

func TestDrainChannel(t *testing.T) {
	ch := make(chan int, 10)

	// Add items to channel
	for i := 0; i < 5; i++ {
		ch <- i
	}

	items := DrainChannel(ch)

	if len(items) != 5 {
		t.Errorf("Expected 5 items, got %d", len(items))
	}

	for i, item := range items {
		if item != i {
			t.Errorf("Expected item %d, got %d", i, item)
		}
	}

	// Should be empty now
	remainingItems := DrainChannel(ch)
	if len(remainingItems) != 0 {
		t.Errorf("Expected 0 remaining items, got %d", len(remainingItems))
	}
}

func TestSafeChannelSend(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := make(chan int, 1)

	// Should succeed
	success := SafeChannelSend(ctx, ch, 42, 100*time.Millisecond)
	if !success {
		t.Error("Expected send to succeed")
	}

	// Should timeout (channel is full)
	success = SafeChannelSend(ctx, ch, 43, 50*time.Millisecond)
	if success {
		t.Error("Expected send to timeout")
	}

	// Should be cancelled
	cancel()
	success = SafeChannelSend(ctx, ch, 44, 100*time.Millisecond)
	if success {
		t.Error("Expected send to be cancelled")
	}
}

func TestBatchProcessor(t *testing.T) {
	items := make([]int, 25)
	for i := 0; i < 25; i++ {
		items[i] = i
	}

	var processedBatches int32

	results := BatchProcessor(
		items,
		10,
		func(batch []int) []int {
			atomic.AddInt32(&processedBatches, 1)

			// Return failed items (odd numbers)
			var failed []int
			for _, item := range batch {
				if item%2 == 1 {
					failed = append(failed, item)
				}
			}
			return failed
		},
	)

	// Should have processed 3 batches (10, 10, 5)
	if processedBatches != 3 {
		t.Errorf("Expected 3 batches, got %d", processedBatches)
	}

	// Should have 12 failed items (odd numbers from 1 to 23)
	if len(results) != 12 {
		t.Errorf("Expected 12 failed items, got %d", len(results))
	}
}

func TestRetryWithBackoff(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	attempts := 0

	// Function that succeeds on 3rd attempt
	fn := func() (string, error) {
		attempts++
		if attempts < 3 {
			return "", fmt.Errorf("attempt %d failed", attempts)
		}
		return "success", nil
	}

	result, err := RetryWithBackoff(
		ctx,
		fn,
		5,
		1*time.Millisecond,
	)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if result != "success" {
		t.Errorf("Expected 'success', got %s", result)
	}

	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

func TestRetryWithBackoff_MaxRetries(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	attempts := 0

	// Function that always fails
	fn := func() (string, error) {
		attempts++
		return "", fmt.Errorf("attempt %d failed", attempts)
	}

	result, err := RetryWithBackoff(
		ctx,
		fn,
		3,
		1*time.Millisecond,
	)

	if err == nil {
		t.Error("Expected error after max retries")
	}

	if result != "" {
		t.Errorf("Expected empty result, got %s", result)
	}

	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}
