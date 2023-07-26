package handlers

import (
	"encoding/json"
	"github.com/Osselnet/metrics-collector/internal/server/db"
	"github.com/Osselnet/metrics-collector/internal/server/middleware/gzip"
	"github.com/Osselnet/metrics-collector/internal/server/middleware/logger"
	"github.com/Osselnet/metrics-collector/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"log"
	"os"
)

type Handler struct {
	router    chi.Router
	Storage   storage.Repositories
	dbStorage db.DateBaseStorage
	key       string
}

func New(router chi.Router, dbStorage db.DateBaseStorage, filename string, restore bool, key string) *Handler {
	h := &Handler{
		router:    router,
		dbStorage: dbStorage,
		key:       key,
	}

	if h.dbStorage != nil {
		h.Storage = h.dbStorage
		log.Println("database storer chosen")
	} else {
		log.Println("default storer chosen")
		h.Storage = storage.New()
	}

	if restore {
		f, err := os.OpenFile(filename, os.O_RDONLY|os.O_CREATE, 0666)
		if err != nil {
			log.Println("File open error", err)
		}
		defer f.Close()

		decoder := json.NewDecoder(f)
		err = decoder.Decode(&h.Storage)
		if err != nil {
			log.Println("Could not restore data", err)
		}
	}

	h.router.Use(middleware.RequestID)
	h.router.Use(middleware.RealIP)
	h.router.Use(middleware.Recoverer)
	h.router.Use(logger.LogHandler)
	h.router.Use(gzip.GzipHandle)

	h.setRoutes()

	return h
}

func (h *Handler) WithStorage(st *storage.MemStorage) {
	h.Storage = st
}

func (h *Handler) setRoutes() {
	h.router.Get("/", h.List)

	//POST http://<АДРЕС_СЕРВЕРА>/update/<ТИП_МЕТРИКИ>/<ИМЯ_МЕТРИКИ>/<ЗНАЧЕНИЕ_МЕТРИКИ>
	h.router.Post("/update/{type}/{name}/{value}", h.Post)

	h.router.Post("/updates/", h.HandleBatchUpdate)

	//GET http://<АДРЕС_СЕРВЕРА>/value/<ТИП_МЕТРИКИ>/<ИМЯ_МЕТРИКИ>
	h.router.Get("/value/{type}/{name}", h.Get)

	h.router.Post("/value/", h.JSONValue)
	h.router.Post("/update/", h.JSONUpdate)

	h.router.Get("/ping", h.Ping)
}

func (h *Handler) GetRouter() chi.Router {
	return h.router
}
