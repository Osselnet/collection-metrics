package handlers

import (
	"github.com/Osselnet/metrics-collector/internal/logger"
	"github.com/Osselnet/metrics-collector/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Handler struct {
	router  chi.Router
	storage *storage.MemStorage
}

func New(router chi.Router, storage *storage.MemStorage) *Handler {
	h := &Handler{
		router:  router,
		storage: storage,
	}

	h.router.Use(middleware.RequestID)
	h.router.Use(middleware.RealIP)
	h.router.Use(middleware.Logger)
	h.router.Use(middleware.Recoverer)

	h.setRoutes()

	return h
}

func (h *Handler) WithStorage(st *storage.MemStorage) {
	h.storage = st
}

func (h *Handler) setRoutes() {
	h.router.Get("/", logger.LogHandler(h.List()))

	//POST http://<АДРЕС_СЕРВЕРА>/update/<ТИП_МЕТРИКИ>/<ИМЯ_МЕТРИКИ>/<ЗНАЧЕНИЕ_МЕТРИКИ>
	h.router.Post("/update/{type}/{name}/{value}", logger.LogHandler(h.Post()))

	//GET http://<АДРЕС_СЕРВЕРА>/value/<ТИП_МЕТРИКИ>/<ИМЯ_МЕТРИКИ>
	h.router.Get("/value/{type}/{name}", logger.LogHandler(h.Get()))
}

func (h *Handler) GetRouter() chi.Router {
	return h.router
}
