package concurrency

import (
	"sync"
	"sync/atomic"
)

// WorkItem represents a work item to be processed by a worker.
// It contains a generic data field.
type WorkItem interface{}

// WorkResult represents the result of a processed work item.
// It contains a generic result field and an error field.
type WorkResult struct {
	Result interface{}
	Err    error
}

// AdaptiveWorkerPool is an adaptive worker pool with backpressure.
// It can scale the number of workers up and down based on the queue size.
// It has a minimum and maximum number of workers.
// It has a request channel, a results channel, and a shutdown channel.
// It uses a wait group to wait for all workers to finish.
type AdaptiveWorkerPool struct {
	workers    int32
	maxWorkers int32
	minWorkers int32
	requests   chan WorkItem
	results    chan WorkResult
	shutdown   chan struct{}
	wg         sync.WaitGroup
}

// NewAdaptiveWorkerPool creates a new AdaptiveWorkerPool.
func NewAdaptiveWorkerPool(minWorkers, maxWorkers, queueDepth int) *AdaptiveWorkerPool {
	pool := &AdaptiveWorkerPool{
		minWorkers: int32(minWorkers),
		maxWorkers: int32(maxWorkers),
		requests:   make(chan WorkItem, queueDepth),
		results:    make(chan WorkResult, queueDepth),
		shutdown:   make(chan struct{}),
	}

	for i := 0; i < int(minWorkers); i++ {
		pool.wg.Add(1)
		go pool.worker()
	}
	atomic.StoreInt32(&pool.workers, int32(minWorkers))

	return pool
}

// worker is the worker function.
// It processes work items from the requests channel.
// It sends results to the results channel.
// It can be shut down by the shutdown channel.
func (p *AdaptiveWorkerPool) worker() {
	defer p.wg.Done()
	for {
		select {
		case req, ok := <-p.requests:
			if !ok {
				return
			}
			// This is where the work is done.
			// In a real implementation, this would be a call to a function that does the work.
			// For now, we just return the request as the result.
			p.results <- WorkResult{Result: req, Err: nil}
		case <-p.shutdown:
			return
		}
	}
}

// adjustWorkers adjusts the number of workers based on the queue size.
// It scales up if the queue is growing and scales down if the queue is empty.
func (p *AdaptiveWorkerPool) adjustWorkers(queueSize int) {
	current := atomic.LoadInt32(&p.workers)

	// Scale up if queue is growing
	if queueSize > cap(p.requests)/2 && current < p.maxWorkers {
		if atomic.CompareAndSwapInt32(&p.workers, current, current+1) {
			p.wg.Add(1)
			go p.worker()
		}
	}

	// Scale down if queue is empty
	if queueSize == 0 && current > p.minWorkers {
		select {
		case p.shutdown <- struct{}{}:
			atomic.AddInt32(&p.workers, -1)
		default:
		}
	}
}

// Submit submits a work item to the pool.
func (p *AdaptiveWorkerPool) Submit(item WorkItem) {
	p.requests <- item
	p.adjustWorkers(len(p.requests))
}

// Results returns the results channel.
func (p *AdaptiveWorkerPool) Results() <-chan WorkResult {
	return p.results
}

// Shutdown shuts down the worker pool.
func (p *AdaptiveWorkerPool) Shutdown() {
	close(p.requests)
	p.wg.Wait()
	close(p.results)
}