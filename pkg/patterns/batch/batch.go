package batch

import (
	"context"
	"sync"
)

// BatchItem represents a single item in a batch operation.
// It contains either a result or an error.
// This allows for partial success in batch operations.
// The original index is preserved for re-ordering results.
// The generic type TResult is the type of the result.
// The generic type TRequest is the type of the request.
type BatchItem[TResult any] struct {
	Result TResult
	Error  error
}

// BatchResult contains the results of a batch operation.
// It is a slice of BatchItem, which contains either a result or an error.
// This allows for partial success in batch operations.
// The original index is preserved for re-ordering results.
// The generic type TResult is the type of the result.
// The generic type TRequest is the type of the request.
type BatchResult[TResult any] struct {
	Items []BatchItem[TResult]
}

// BatchProcessor is a generic batch processor.
// It can be used to process batches of requests concurrently.
// It supports a maximum concurrency level and can continue on error.
// The generic type TRequest is the type of the request.
// The generic type TResult is the type of the result.
// The validate function is used to validate a request before processing.
// The process function is used to process a request.
// The MaxConcurrency field is the maximum number of concurrent requests.
// The continueOnError field determines whether to continue processing on error.
type BatchProcessor[TRequest, TResult any] struct {
	MaxConcurrency int
	Validate      func(TRequest) error
	Process       func(context.Context, TRequest) (TResult, error)
}

// ProcessBatch processes a batch of requests.
// It returns a BatchResult containing the results of the operation.
// It supports a maximum concurrency level and can continue on error.
// The generic type TRequest is the type of the request.
// The generic type TResult is the type of the result.
// The Validate function is used to validate a request before processing.
// The Process function is used to process a request.
// The MaxConcurrency field is the maximum number of concurrent requests.
// The continueOnError field determines whether to continue processing on error.
func (bp *BatchProcessor[TRequest, TResult]) ProcessBatch(
	ctx context.Context, 
	requests []TRequest, 
	continueOnError bool,
) (*BatchResult[TResult], error) {
	results := make([]BatchItem[TResult], len(requests))
	semaphore := make(chan struct{}, bp.MaxConcurrency)
	
	var wg sync.WaitGroup
	for i, req := range requests {
		wg.Add(1)
		go func(index int, request TRequest) {
			defer wg.Done()
			semaphore <- struct{}{} // Acquire
			defer func() { <-semaphore }() // Release
			
			if err := bp.Validate(request); err != nil {
				results[index] = BatchItem[TResult]{Error: err}
				return
			}
			
			result, err := bp.Process(ctx, request)
			results[index] = BatchItem[TResult]{Result: result, Error: err}
		}(i, req)
	}
	
	wg.Wait()
	return &BatchResult[TResult]{Items: results}, nil
}
