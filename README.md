# Worker

The purpose of this library is to manage a queue of work and utilize `N` workers (via goroutines) to process the tasks concurrently. The queue has a specified size, and when adding a task to the queue, it will block until a worker has completed a task and freed up space in the queue.

## Features

- Concurrently processes tasks using multiple workers (goroutines).
- Fixed-size queue to manage tasks.
- Blocks when adding tasks if the queue is full.

## Limitations

This library does not currently support:
- Timeout/context for a worker.
- Retry mechanism.

## Usage

Here are some examples to demonstrate how to use the Worker library:

```bash
go get github.com/jtarchie/worke
```

### Basic Usage

```go
count := int32(0)
w := worker.New[int](1, 1, func(index, value int) {
	atomic.AddInt32(&count, 1)
})
defer w.Close()
w.Enqueue(100)
```

### Handling Panics in Workers

Ensure your application remains robust even when panics occur within a worker.

```go
w := worker.New[int](10, 1, func(index, value int) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered in worker:", r)
		}
	}()

	if value == 100 {
		panic("a problem has entered the chat")
	}
})
defer w.Close()
w.Enqueue(100)
```

### Distributing Work Across Workers

If you have multiple tasks to be processed, you can distribute them across different workers.

```go
count, workers := int32(0), make(chan int, 10)
ctx, cancel := context.WithCancel(context.Background())

w := worker.New[int](1, 10, func(index, value int) {
	workers <- index
	atomic.AddInt32(&count, 1)
	for range ctx.Done() {}
})
defer w.Close()
defer cancel()

for i := 0; i < 10; i++ {
	w.Enqueue(i)
}
```

### Handling Large Amounts of Work

For processing a large number of tasks with different queue and worker configurations:

```go
w := worker.New[int](10, 10, func(index, value int) {
	atomic.AddInt32(&count, 1)
})
defer w.Close()

for i := 0; i < 100_000; i++ {
	w.Enqueue(i)
}
```

For more intricate examples and edge cases, please refer to the provided tests.
