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
		go w.startWorker(index)
	}
}

func (w *Worker[T]) startWorker(index int) {
	w.wait.Add(1)
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
	enqueueSettings := &enqueueOption{}
	for _, option := range options {
		option(enqueueSettings)
	}

	if enqueueSettings.timeout == 0 {
		w.queue <- item

		return true
	}

	expiration := time.After(enqueueSettings.timeout)
	select {
	case <-expiration:
		return false
	case w.queue <- item:
		return true
	}
}

func (w *Worker[T]) Close() {
	runtime.Gosched()
	close(w.queue)
	w.wait.Wait()
}
