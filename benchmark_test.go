package worker_test

import (
	"runtime"
	"sync/atomic"
	"testing"

	"github.com/jtarchie/worker"
)

func BenchmarkEntries(b *testing.B) {
	b.Run("1,1", func(b *testing.B) {
		var count atomic.Int32
		pool := worker.New[int](1, 1, func(index, value int) {
			count.Add(1)
		})
		defer pool.Close()

		b.ResetTimer()

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				pool.Enqueue(1)
			}
		})
	})

	b.Run("N,N", func(b *testing.B) {
		var count atomic.Int32
		pool := worker.New[int](runtime.NumCPU(), runtime.NumCPU(), func(index, value int) {
			count.Add(1)
		})
		defer pool.Close()

		b.ResetTimer()

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				pool.Enqueue(1)
			}
		})
	})
}
