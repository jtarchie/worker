package worker

import (
	"runtime"
	"sync"
	"time"
)

type Worker[T any] struct {
	process func(int, T)
	queue   chan T
	size    int
	wait    *sync.WaitGroup
}

func New[T any](
	queueSize int,
	workerSize int,
	process func(int, T),
) *Worker[T] {
	pool := &Worker[T]{
		process: process,
		queue:   make(chan T, queueSize),
		size:    workerSize,
		wait:    &sync.WaitGroup{},
	}
	go pool.init()

	return pool
}

func (w *Worker[T]) init() {
	for index := 1; index <= w.size; index++ {
		w.wait.Add(1)
		go w.startWorker(index)
	}
}

func (w *Worker[T]) startWorker(index int) {
	defer w.wait.Done()

	for item := range w.queue {
		w.process(index, item)
	}
}

type enqueueOption struct {
	timeout time.Duration
}

type WithOption func(*enqueueOption)

func WithTimeout(duration time.Duration) WithOption {
	return func(e *enqueueOption) {
		e.timeout = duration
	}
}

func (w *Worker[T]) Enqueue(item T, options ...WithOption) bool {
	const theFuture = time.Hour * 24 * 365 * 100 // 100 years in the future
	enqueueSettings := &enqueueOption{
		timeout: theFuture,
	}

	for _, option := range options {
		option(enqueueSettings)
	}

	expiration := time.NewTimer(enqueueSettings.timeout)
	defer expiration.Stop()

	select {
	case <-expiration.C:
		return false
	case w.queue <- item:
		return true
	}
}

func (w *Worker[T]) Close() {
	// this is to prevent exhaustion from init()
	// with an immediate `defer w.Close()` that might be done
	runtime.Gosched()
	close(w.queue)
	w.wait.Wait()
}
