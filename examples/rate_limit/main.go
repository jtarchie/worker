// You can edit this code!
// Click here and start typing.
package main

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/jtarchie/worker"
)

func main() {
	type Result struct {
		response *http.Response
		error error
	}

	consumer := worker.New[Result](10, 10, func(index int, result Result) {
		slog.Info("http response", slog.Any("result", result), slog.Int("worker", index))
	})

	producer := worker.New[string](10,1, func(index int, url string) {
		slog.Info("getting URL", slog.String("url", url))
		
		waiter := time.After(time.Second)
		
		response, err := http.Get(url)
		consumer.Enqueue(Result{
			response: response,
			error: err,
		})

		<-waiter
	})
	defer consumer.Close()
	defer producer.Close()
	
	for i := 0; i < 10; i++ {
		producer.Enqueue("https://example.com")
	}
}
