package main

import (
	"context"
	"github.com/Osselnet/metrics-collector/internal/server/config"
	"github.com/Osselnet/metrics-collector/internal/server/handlers"
	"github.com/Osselnet/metrics-collector/internal/storage"
	"github.com/Osselnet/metrics-collector/pkg/metrics"
	"github.com/go-chi/chi/v5"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var addr string

func main() {
	cfg, err := config.ParseConfig()
	if err != nil {
		panic(err)
	}

	storage := &storage.MemStorage{
		Metrics: metrics.New(),
	}

	go func() {
		for {
			time.Sleep(time.Second * time.Duration(cfg.Interval))
			storage.WriteDataToFile(cfg.Filename)
		}
	}()

	h := handlers.New(chi.NewRouter(), storage, cfg.Filename, cfg.Restore)
	server := http.Server{
		Addr:    cfg.Address,
		Handler: h.GetRouter(),
	}

	idleConnectionsClosed := make(chan struct{})
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
		<-sigint
		log.Println("Shutting down server")

		if err := storage.WriteDataToFile(cfg.Filename); err != nil {
			log.Printf("Error during saving data to file: %v", err)
		}

		if err := server.Shutdown(context.Background()); err != nil {
			log.Printf("HTTP Server Shutdown Error: %v", err)
		}
		close(idleConnectionsClosed)
	}()

	err = server.ListenAndServe()
	if err != nil {
		panic(err)
	}
}
