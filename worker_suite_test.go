package worker_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jtarchie/worker"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestWorker(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Worker Suite")
}

var _ = Describe("Worker", func() {
	It("can process work", func() {
		var count atomic.Int32
		pool := worker.New[int](1, 1, func(index, value int) {
			defer GinkgoRecover()

			Expect(index).To(Equal(1))
			Expect(value).To(Equal(100))

			count.Add(1)
		})
		defer pool.Close()

		Expect(pool.Enqueue(100)).To(BeTrue())

		Eventually(count.Load).Should(BeEquivalentTo(1))
	})

	When("a panic happens on a worker", func() {
		It("does not bring anything down", func() {
			var (
				count      atomic.Int32
				recoveries atomic.Int32
			)
			pool := worker.New[int](10, 1, func(index, value int) {
				defer func() {
					if r := recover(); r != nil {
						recoveries.Add(1)
					}
				}()

				count.Add(1)

				if value == 100 {
					panic("a problem has entered the chat")
				}
			})
			defer pool.Close()

			Expect(pool.Enqueue(100)).To(BeTrue())
			Expect(pool.Enqueue(101)).To(BeTrue())

			Eventually(count.Load).Should(BeEquivalentTo(2))
			Eventually(recoveries.Load).Should(BeEquivalentTo(1))
		})
	})

	It("can process work across workers", func() {
		var count atomic.Int32
		workers := make(chan int, 10)
		pool := worker.New[int](1, 10, func(index, value int) {
			// keep track of how many workers have been used
			workers <- index

			// count how many times this function is called
			count.Add(1)

			// block so other, this worker does not pick up more work
			time.Sleep(time.Millisecond)
		})
		defer pool.Close()

		go func() {
			for i := range 10 {
				Expect(pool.Enqueue(i)).To(BeTrue())
			}
		}()

		Eventually(count.Load).Should(BeEquivalentTo(10))

		close(workers)

		usedWorkers := []int{}
		for id := range workers {
			usedWorkers = append(usedWorkers, id)
		}

		Expect(usedWorkers).To(HaveLen(10))
		Expect(usedWorkers).To(ContainElements(1, 2, 3, 4, 5, 6, 7, 8, 9, 10))
	})

	DescribeTable("handling a lot of work", func(queueSize, workerSize, elements int) {
		var count atomic.Int32
		pool := worker.New[int](queueSize, workerSize, func(index, value int) {
			count.Add(1)
		})
		defer pool.Close()

		go func() {
			for i := range elements {
				Expect(pool.Enqueue(i)).To(BeTrue())
			}
		}()

		Eventually(count.Load).Should(BeEquivalentTo(elements))
	},
		Entry("1,1,100", 1, 1, 100),
		Entry("1,1,1000", 1, 1, 100),
		Entry("10,1,1000", 10, 1, 1_000),
		Entry("1,10,1000", 10, 1, 1_000),
		Entry("10,10,1000", 10, 10, 1_000),
		Entry("10,10,1000", 10, 10, 100_000),
	)

	When("providing a timeout", func() {
		It("can handle a timeout for queueing", func() {
			var neverGetHere atomic.Bool
			neverGetHere.Store(true)

			pool := worker.New[int](0, 0, func(index, value int) {
				neverGetHere.Store(false)
			})
			defer pool.Close()

			Expect(pool.Enqueue(1, worker.WithTimeout(time.Millisecond))).To(BeFalse())

			Consistently(neverGetHere.Load).Should(BeTrue())
		})

		It("will not queue before the timeout", func() {
			var count atomic.Int32
			pool := worker.New[int](1, 1, func(index, value int) {
				count.Add(1)
				time.Sleep(time.Second)
			})
			defer pool.Close()

			Eventually(func() bool {
				return pool.Enqueue(1, worker.WithTimeout(time.Millisecond))
			}).Should(BeFalse())

			Eventually(count.Load).Should(BeEquivalentTo(1))
		})
	})

	When("using context", func() {
		It("can create a worker with context", func() {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			var count atomic.Int32
			pool := worker.NewWithContext[int](ctx, 1, 1, func(index, value int) {
				count.Add(1)
			})
			defer pool.Close()

			Expect(pool.Enqueue(100)).To(BeTrue())
			Eventually(count.Load).Should(BeEquivalentTo(1))
		})

		It("stops workers when context is cancelled", func() {
			ctx, cancel := context.WithCancel(context.Background())

			var count atomic.Int32
			pool := worker.NewWithContext[int](ctx, 10, 2, func(index, value int) {
				count.Add(1)
				time.Sleep(50 * time.Millisecond)
			})

			// Enqueue some work
			for i := range 5 {
				Expect(pool.Enqueue(i)).To(BeTrue())
			}

			// Wait for some processing
			Eventually(count.Load).Should(BeNumerically(">=", 1))

			// Cancel context
			cancel()

			// Close should complete quickly
			done := make(chan struct{})
			go func() {
				pool.Close()
				close(done)
			}()

			Eventually(done, 5*time.Second).Should(BeClosed())
		})

		It("can use WithContext option for enqueue", func() {
			pool := worker.New[int](0, 0, func(index, value int) {
				// never called
			})
			defer pool.Close()

			ctx, cancel := context.WithCancel(context.Background())
			cancel() // Cancel immediately

			// Enqueue should fail because context is cancelled
			Expect(pool.Enqueue(1, worker.WithContext(ctx))).To(BeFalse())
		})

		It("provides Done channel", func() {
			ctx, cancel := context.WithCancel(context.Background())
			pool := worker.NewWithContext[int](ctx, 1, 1, func(index, value int) {})

			// Done channel should not be closed yet
			select {
			case <-pool.Done():
				Fail("Done channel should not be closed")
			default:
				// expected
			}

			cancel()

			// Done channel should be closed after cancel
			Eventually(pool.Done()).Should(BeClosed())

			pool.Close()
		})
	})
})
