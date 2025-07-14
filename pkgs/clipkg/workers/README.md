# SimpleWorker - Generic Producer-Consumer Utility

The SimpleWorker package provides a clean, minimal, and reusable producer-consumer pattern that's generic and deadlock-resistant.

## Features

- **Generic**: Works with any type of work items and results using Go generics
- **Deadlock-resistant**: Uses channels and atomic operations instead of condition variables
- **Error handling**: Proper error propagation and failed item collection
- **Statistics**: Built-in tracking of produced, consumed, and failed items
- **Utilities**: Additional helper functions for common patterns

## Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/WangWilly/xSync/pkgs/workers"
)

type WorkItem struct {
    ID   int
    Data string
}

type FailedItem struct {
    ID    int
    Error string
}

func main() {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // Create a simple worker with 3 consumer workers
    worker := workers.NewSimpleWorker[WorkItem, FailedItem](ctx, cancel, 3)

    // Define a producer that generates work items
    producer := func(ctx context.Context, output chan<- WorkItem) error {
        for i := 0; i < 10; i++ {
            select {
            case <-ctx.Done():
                return ctx.Err()
            case output <- WorkItem{ID: i, Data: fmt.Sprintf("item-%d", i)}:
                worker.IncrementProduced()
            }
        }
        return nil
    }

    // Define a consumer that processes work items
    consumer := func(ctx context.Context, input <-chan WorkItem) []FailedItem {
        var failed []FailedItem
        
        for {
            select {
            case <-ctx.Done():
                return failed
            case item, ok := <-input:
                if !ok {
                    return failed
                }
                
                worker.IncrementConsumed()
                
                // Process item here
                if err := processItem(item); err != nil {
                    failed = append(failed, FailedItem{
                        ID:    item.ID,
                        Error: err.Error(),
                    })
                }
            }
        }
    }

    // Run the producer-consumer pipeline
    result := worker.Process(producer, consumer, 5)

    // Handle results
    fmt.Printf("Produced: %d, Consumed: %d, Failed: %d, Duration: %v\n",
        result.Stats.Produced, result.Stats.Consumed, result.Stats.Failed, result.Stats.Duration)

    if result.Error != nil {
        fmt.Printf("Producer error: %v\n", result.Error)
    }

    for _, failed := range result.Failed {
        fmt.Printf("Failed item %d: %s\n", failed.ID, failed.Error)
    }
}
```

## API Reference

### SimpleWorker[T, R]

Generic producer-consumer worker where:
- `T` is the type of work items
- `R` is the type of failed/result items

#### Methods

- `NewSimpleWorker[T, R](ctx, cancel, maxWorkers)` - Creates a new worker
- `Process(producer, consumer, bufferSize)` - Runs the producer-consumer pipeline
- `IncrementProduced()` - Atomically increments produced counter
- `IncrementConsumed()` - Atomically increments consumed counter
- `GetStats()` - Returns current processing statistics

#### Types

- `ProducerFunc[T]` - Function that generates work items
- `ConsumerFunc[T, R]` - Function that processes work items and returns failed items
- `ProcessResult[R]` - Contains results, errors, and statistics
- `ProcessStats` - Contains processing statistics

## Utility Functions

### DrainChannel[T]

Safely drains all remaining items from a channel:

```go
items := workers.DrainChannel(workChan)
```

### SafeChannelSend[T]

Attempts to send to a channel with timeout and context cancellation:

```go
success := workers.SafeChannelSend(ctx, ch, item, 1*time.Second)
```

### BatchProcessor[T, R]

Processes items in batches:

```go
results := workers.BatchProcessor(
    items,
    10, // batch size
    func(batch []Item) []Result {
        // Process batch
        return results
    },
)
```

### RetryWithBackoff[T]

Retries a function with exponential backoff:

```go
result, err := workers.RetryWithBackoff(
    ctx,
    func() (string, error) {
        // Your operation here
        return "success", nil
    },
    5,                      // max retries
    100*time.Millisecond,   // initial delay
)
```

## Design Principles

1. **No Deadlocks**: Uses channels and atomic operations instead of condition variables
2. **Clear Ownership**: Producers own the work channel, consumers return failed items
3. **Proper Cleanup**: All channels are properly closed and drained
4. **Error Handling**: Both producer errors and consumer failures are collected
5. **Statistics**: Built-in tracking for monitoring and debugging

## Thread Safety

- All operations are thread-safe
- Uses atomic operations for counters
- Proper synchronization with WaitGroups
- Mutex protection for shared data structures

## Error Handling

- Producer errors are captured and returned in `ProcessResult.Error`
- Consumer failures are returned as items in `ProcessResult.Failed`
- Context cancellation is properly handled throughout
- No silent failures or lost work items

## Performance Considerations

- Use appropriate buffer sizes for the work channel
- Consider batch processing for high-throughput scenarios
- Monitor the statistics to tune worker count and buffer sizes
- The utility functions are optimized for common patterns

## Examples

See `example.go` for comprehensive examples of:
- Basic producer-consumer usage
- Retry with exponential backoff
- Batch processing
- Channel draining
- Safe channel sending

## Migration from Original Worker

If you're migrating from the original tweet media download worker, the SimpleWorker provides the same deadlock-resistant patterns in a more generic form. The key differences:

1. Generic types instead of specific tweet types
2. Consumers return failed items instead of using error channels
3. Statistics are built-in and atomic
4. Utilities are provided as separate functions
5. More flexible and reusable for different use cases
