package worker

import (
	"sync"
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
	w := &Worker[T]{
		process: process,
		queue:   make(chan T, queueSize),
		size:    workerSize,
		wait:    &sync.WaitGroup{},
	}
	go w.init()

	return w
}

func (w *Worker[T]) init() {
	for index := 1; index <= w.size; index++ {
		go w.startWorker(index)
	}
}

func (w *Worker[T]) startWorker(index int) {
	w.wait.Add(1)

	for item := range w.queue {
		w.process(index, item)
	}

	w.wait.Done()
}

func (w *Worker[T]) Enqueue(item T) {
	w.queue <- item
}

func (w *Worker[T]) Close() {
	close(w.queue)
	w.wait.Wait()
}
