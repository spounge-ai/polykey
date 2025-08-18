package concurrency

import (
	"context"
	"sync"
)

// Job represents a work item to be processed, including the data.
type Job[T any] struct {
	Data T
}

// Result holds the outcome of a processed job.
type Result[R any] struct {
	Value R
	Err   error
}

// Processor is a function that processes a single job.
type Processor[T, R any] func(ctx context.Context, data T) (R, error)

// WorkerPool is a generic, fixed-size worker pool for concurrent task processing.
// It is type-safe and uses a clear start/stop lifecycle.
type WorkerPool[T, R any] struct {
	workerCount int
	jobs        chan Job[T]
	results     chan Result[R]
	processor   Processor[T, R]
	wg          sync.WaitGroup
	ctx         context.Context
	cancel      context.CancelFunc
}

// NewWorkerPool creates a new fixed-size worker pool.
// - parentCtx: The context to govern the lifecycle of the pool's workers.
// - workerCount: The number of concurrent workers to run.
// - queueDepth: The buffer size of the job queue.
// - processor: The function that will be executed by each worker for each job.
func NewWorkerPool[T, R any](parentCtx context.Context, workerCount int, queueDepth int, processor Processor[T, R]) *WorkerPool[T, R] {
	ctx, cancel := context.WithCancel(parentCtx)
	return &WorkerPool[T, R]{
		workerCount: workerCount,
		jobs:        make(chan Job[T], queueDepth),
		results:     make(chan Result[R], queueDepth),
		processor:   processor,
		ctx:         ctx,
		cancel:      cancel,
	}
}

// Start initializes and starts the workers in the pool.
func (p *WorkerPool[T, R]) Start() {
	for i := 0; i < p.workerCount; i++ {
		p.wg.Add(1)
		go p.worker()
	}
}

// worker is the core processing loop for a single worker goroutine.
func (p *WorkerPool[T, R]) worker() {
	defer p.wg.Done()
	for {
		select {
		case job, ok := <-p.jobs:
			if !ok {
				// Jobs channel was closed.
				return
			}
			value, err := p.processor(p.ctx, job.Data)
			p.results <- Result[R]{Value: value, Err: err}
		case <-p.ctx.Done():
			// Context was cancelled, terminate worker.
			return
		}
	}
}

// Submit sends a new job to the pool for processing.
// This is a non-blocking call. If the job queue is full, the job is dropped.
// Returns true if the job was submitted, false if the queue was full.
func (p *WorkerPool[T, R]) Submit(data T) bool {
	select {
	case p.jobs <- Job[T]{Data: data}:
		return true
	default:
		return false
	}
}

// Results returns the channel from which processed results can be read.
func (p *WorkerPool[T, R]) Results() <-chan Result[R] {
	return p.results
}

// Stop gracefully shuts down the worker pool.
// It waits for all submitted jobs to be processed before returning.
func (p *WorkerPool[T, R]) Stop() {
	// Close the jobs channel to signal workers to stop after finishing current work.
	close(p.jobs)
	// Wait for all worker goroutines to finish.
	p.wg.Wait()
	// Close the results channel after all workers are done.
	close(p.results)
	// Cancel the context to clean up any resources.
	p.cancel()
}