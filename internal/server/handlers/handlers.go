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
	storage   *storage.MemStorage
	dbStorage db.DBStorage
}

func New(router chi.Router, storage *storage.MemStorage, dbStorage db.DBStorage, filename string, restore bool) *Handler {
	h := &Handler{
		router:    router,
		storage:   storage,
		dbStorage: dbStorage,
	}

	if restore {
		f, err := os.OpenFile(filename, os.O_RDONLY|os.O_CREATE, 0666)
		if err != nil {
			log.Println("File open error", err)
		}
		defer f.Close()

		decoder := json.NewDecoder(f)
		err = decoder.Decode(&h.storage)
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
	h.storage = st
}

func (h *Handler) setRoutes() {
	h.router.Get("/", h.List)

	//POST http://<АДРЕС_СЕРВЕРА>/update/<ТИП_МЕТРИКИ>/<ИМЯ_МЕТРИКИ>/<ЗНАЧЕНИЕ_МЕТРИКИ>
	h.router.Post("/update/{type}/{name}/{value}", h.Post)

	//GET http://<АДРЕС_СЕРВЕРА>/value/<ТИП_МЕТРИКИ>/<ИМЯ_МЕТРИКИ>
	h.router.Get("/value/{type}/{name}", h.Get)

	h.router.Post("/value/", h.JSONValue)
	h.router.Post("/update/", h.JSONUpdate)

	h.router.Get("/ping", h.Ping)
}

func (h *Handler) GetRouter() chi.Router {
	return h.router
}
