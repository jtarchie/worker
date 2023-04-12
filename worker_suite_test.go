package worker_test

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"

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
		count := int32(0)
		w := worker.New[int](1, 1, func(index, value int) {
			defer GinkgoRecover()

			Expect(index).To(Equal(1))
			Expect(value).To(Equal(100))

			atomic.AddInt32(&count, 1)
		})
		defer w.Close()

		w.Enqueue(100)

		Eventually(func() int32 {
			return atomic.LoadInt32(&count)
		}).Should(BeEquivalentTo(1))
	})

	When("a panic happens on a worker", func() {
		It("does not bring anything down", func() {
			count := int32(0)
			w := worker.New[int](10, 1, func(index, value int) {
				defer func() {
					if r := recover(); r != nil {
						fmt.Println("Recovered in f", r)
					}
				}()

				atomic.AddInt32(&count, 1)

				if value == 100 {
					panic("a problem has entered the chat")
				}
			})
			defer w.Close()

			w.Enqueue(100)
			w.Enqueue(101)

			Eventually(func() int32 {
				return atomic.LoadInt32(&count)
			}).Should(BeEquivalentTo(2))
		})
	})

	It("can process work across workers", func() {
		count, workers := int32(0), make(chan int, 10)
		ctx, cancel := context.WithCancel(context.Background())

		w := worker.New[int](1, 10, func(index, value int) {
			// keep track of how many workers have been used
			workers <- index

			// count how many times this function is called
			atomic.AddInt32(&count, 1)

			// block so other, this worker does not pick up more work
			for range ctx.Done() {
			}
		})
		defer w.Close()
		defer cancel()

		go func() {
			for i := 0; i < 10; i++ {
				w.Enqueue(i)
			}
		}()

		Eventually(func() int32 {
			return atomic.LoadInt32(&count)
		}).Should(BeEquivalentTo(10))

		cancel()
		close(workers)

		usedWorkers := []int{}
		for id := range workers {
			usedWorkers = append(usedWorkers, id)
		}

		Expect(usedWorkers).To(HaveLen(10))
		Expect(usedWorkers).To(ContainElements(1, 2, 3, 4, 5, 6, 7, 8, 9, 10))
	})

	DescribeTable("handling a lot of work", func(queueSize, workerSize, elements int) {
		count := int32(0)
		w := worker.New[int](queueSize, workerSize, func(index, value int) {
			atomic.AddInt32(&count, 1)
		})
		defer w.Close()

		go func() {
			for i := 0; i < elements; i++ {
				w.Enqueue(i)
			}
		}()

		Eventually(func() int32 {
			return atomic.LoadInt32(&count)
		}).Should(BeEquivalentTo(elements))
	},
		Entry("1,1,100", 1, 1, 100),
		Entry("1,1,1000", 1, 1, 100),
		Entry("10,1,1000", 10, 1, 1_000),
		Entry("1,10,1000", 10, 1, 1_000),
		Entry("10,10,1000", 10, 10, 1_000),
		Entry("10,10,1000", 10, 10, 100_000),
	)
})
