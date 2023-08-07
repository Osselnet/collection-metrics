package main

import (
	"context"
	"github.com/Osselnet/metrics-collector/internal/server/config"
	"github.com/Osselnet/metrics-collector/internal/server/db"
	"github.com/Osselnet/metrics-collector/internal/server/handlers"
	"github.com/Osselnet/metrics-collector/internal/storage"
	"github.com/go-chi/chi/v5"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	cfg, err := config.ParseConfig()
	if err != nil {
		panic(err)
	}

	var dbStorage db.DateBaseStorage

	if cfg.DSN != "" {
		dbStorage = db.New(cfg.DSN)
	}

	h := handlers.New(chi.NewRouter(), dbStorage, cfg.Filename, cfg.Restore, cfg.Key)
	server := http.Server{
		Addr:    cfg.Address,
		Handler: h.GetRouter(),
	}

	go func() {
		if cfg.DSN == "" && cfg.Filename != "" {
			for {
				time.Sleep(time.Second * time.Duration(cfg.Interval))
				h.Storage.(*storage.MemStorage).WriteDataToFile(cfg.Filename)
			}
		}
	}()

	idleConnectionsClosed := make(chan struct{})
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
		<-sigint
		log.Println("Shutting down server")

		if cfg.DSN == "" && cfg.Filename != "" {
			if err := h.Storage.(*storage.MemStorage).WriteDataToFile(cfg.Filename); err != nil {
				log.Printf("Error during saving data to file: %v", err)
			}
		}

		if dbStorage != nil {
			err := dbStorage.Shutdown()
			if err != nil {
				log.Printf("Database shutdown error: %v", err)
			}
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
