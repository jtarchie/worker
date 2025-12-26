package worker

import (
	"context"
	"sync"
	"time"
)

type Worker[T any] struct {
	process func(int, T)
	queue   chan T
	wait    *sync.WaitGroup
	ctx     context.Context
	cancel  context.CancelFunc
}

// New creates a new worker pool. Use Close() to stop all workers.
func New[T any](
	queueSize int,
	workerSize int,
	process func(int, T),
) *Worker[T] {
	return NewWithContext(context.Background(), queueSize, workerSize, process)
}

// NewWithContext creates a new worker pool with context support.
// Workers will stop when the context is cancelled or Close() is called.
func NewWithContext[T any](
	ctx context.Context,
	queueSize int,
	workerSize int,
	process func(int, T),
) *Worker[T] {
	ctx, cancel := context.WithCancel(ctx)

	pool := &Worker[T]{
		process: process,
		queue:   make(chan T, queueSize),
		wait:    &sync.WaitGroup{},
		ctx:     ctx,
		cancel:  cancel,
	}

	// Start workers synchronously to avoid race between init and Close
	for index := 1; index <= workerSize; index++ {
		pool.wait.Add(1)
		go pool.startWorker(index)
	}

	return pool
}

func (w *Worker[T]) startWorker(index int) {
	defer w.wait.Done()

	for {
		select {
		case <-w.ctx.Done():
			// Context cancelled, drain remaining items and exit
			for item := range w.queue {
				w.process(index, item)
			}
			return
		case item, ok := <-w.queue:
			if !ok {
				return
			}
			w.process(index, item)
		}
	}
}

type enqueueOption struct {
	timeout time.Duration
	ctx     context.Context
}

type WithOption func(*enqueueOption)

// WithTimeout sets a timeout for the enqueue operation.
func WithTimeout(duration time.Duration) WithOption {
	return func(e *enqueueOption) {
		e.timeout = duration
	}
}

// WithContext sets a context for the enqueue operation.
// The enqueue will fail if the context is cancelled.
func WithContext(ctx context.Context) WithOption {
	return func(e *enqueueOption) {
		e.ctx = ctx
	}
}

func (w *Worker[T]) Enqueue(item T, options ...WithOption) bool {
	const theFuture = time.Hour * 24 * 365 * 100 // 100 years in the future
	enqueueSettings := &enqueueOption{
		timeout: theFuture,
		ctx:     nil,
	}

	for _, option := range options {
		option(enqueueSettings)
	}

	expiration := time.NewTimer(enqueueSettings.timeout)
	defer expiration.Stop()

	// If context is provided, also listen for its cancellation
	if enqueueSettings.ctx != nil {
		select {
		case <-enqueueSettings.ctx.Done():
			return false
		case <-expiration.C:
			return false
		case w.queue <- item:
			return true
		}
	}

	select {
	case <-expiration.C:
		return false
	case w.queue <- item:
		return true
	}
}

// Close stops all workers and waits for them to finish processing.
// It closes the queue and waits for all workers to drain remaining items.
func (w *Worker[T]) Close() {
	w.cancel()
	close(w.queue)
	w.wait.Wait()
}

// Done returns a channel that is closed when the worker's context is cancelled.
func (w *Worker[T]) Done() <-chan struct{} {
	return w.ctx.Done()
}
