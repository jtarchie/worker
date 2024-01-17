// You can edit this code!
// Click here and start typing.
package main

import (
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/jtarchie/worker"
)

func main() {
	var locker sync.Mutex

	pool := worker.New[string](10, 10, func(index int, url string) {
		locker.Lock()
		slog.Info("getting URL", slog.String("url", url))

		waiter := time.After(time.Second)

		response, _ := http.Get(url)

		<-waiter
		locker.Unlock()

		slog.Info("http response", slog.Any("result", response), slog.Int("worker", index))

	})

	for i := 0; i < 10; i++ {
		pool.Enqueue("https://example.com")
	}

	pool.Close()
}
