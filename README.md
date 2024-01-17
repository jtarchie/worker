# Worker

The purpose of this library is to manage a queue of work and utilize `N` workers
(via goroutines) to process the tasks concurrently. The queue has a specified
size, and when adding a task to the queue, it will block until a worker has
completed a task and freed up space in the queue.

## Features

- Concurrently processes tasks using multiple workers (goroutines).
- Fixed-size queue to manage tasks.
- Blocks when adding tasks if the queue is full.
- Specify an optional timeout for work to be queued.

### Limitations

This library does not currently support:

- `context` for a worker.
- `context` for a enqueuing.
- Retry mechanism.
- Does not return a value or error

## Usage

Here is how to use the library:

```bash
go get github.com/jtarchie/worker
```

Within in code.

```go
count := int32(0)
pool := worker.New[int](1, 1, func(index, value int) {
	atomic.AddInt32(&count, 1)
})
defer pool.Close()

pool.Enqueue(100)
```

This enqueues a single value for a worker to process, incrementing the count
each time a worker processes a task.

The type int indicates the kind of value that can be enqueued. If a different
value type is passed to Enqueue, the compiler will report an error.

```go
// this results in a compiler type error
pool.Enqueue("some string")
```

The three arguments for `worker.New` are:

1. Buffered queue size determines how many values can be enqueued without
   blocking. Once this buffer is filled, any call to Enqueue will be blocked
   until there's space in the queue.
2. Number of workers in the pool: Corresponds to one worker per goroutine. These
   are cleaned up upon calling Close, ensuring that the running workers process
   all tasks in the queue.
3. Function to process the enqueued value:
   - `index` is the worker being run on. `value` is the enqueued item of type T
     passed to the generic.

You can fine-tune these values for your pool:

```go
pool := worker.New(
	100, // buffered queue is size 100
	10,  // number of goroutines to distribute work across 
	func(index int, value T) {
		// do some work on `value`,
		// T is the generic type
	},
)
```

It's essential to wait for your worker pool to complete by calling Close. This
ensures that workers finalize their tasks and shut down correctly. Using the
defer pattern is highly recommended.
