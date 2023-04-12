# Worker

The purpose of this library is to manage a queue of work and utilize `N` workers
(via goroutines) to process the tasks concurrently. The queue has a specified
size, and when adding a task to the queue, it will block until a worker has
completed a task and freed up space in the queue.

## Features

- Concurrently processes tasks using multiple workers (goroutines)
- Fixed-size queue to manage tasks
- Blocks when adding tasks if the queue is full

## Limitations

This library does not currently support:

- Timeout/context for a worker
- Retry mechanism

For usage examples, please refer to the provided tests.
