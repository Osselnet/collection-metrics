package main

import (
	"flag"
	"github.com/Osselnet/metrics-collector/internal/server/handlers"
	"github.com/Osselnet/metrics-collector/internal/storage"
	"github.com/Osselnet/metrics-collector/pkg/metrics"
	"github.com/go-chi/chi/v5"
	"net/http"
	"os"
)

var addr string

func main() {
	flag.StringVar(&addr, "a", "127.0.0.1:8080", "address to listen on")
	flag.Parse()

	if a := os.Getenv("ADDRESS"); a != "" {
		addr = a
	}

	storage := &storage.MemStorage{
		Metrics: metrics.New(),
	}

	h := handlers.New(chi.NewRouter(), storage)
	err := http.ListenAndServe(addr, h.GetRouter())
	if err != nil {
		panic(err)
	}
}
