package handlers

import (
	"bytes"
	"fmt"
	"github.com/Osselnet/metrics-collector/internal/storage"
	"github.com/Osselnet/metrics-collector/pkg/metrics"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"net/http"
	"sort"
	"strconv"
)

type Handler struct {
	router  chi.Router
	storage *storage.MemStorage
}

func New() *Handler {
	h := &Handler{
		router:  chi.NewRouter(),
		storage: storage.New(),
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
	h.router.Get("/", h.List)

	//POST http://<АДРЕС_СЕРВЕРА>/update/<ТИП_МЕТРИКИ>/<ИМЯ_МЕТРИКИ>/<ЗНАЧЕНИЕ_МЕТРИКИ>
	h.router.Post("/update/{type}/{name}/{value}", h.Post)

	//GET http://<АДРЕС_СЕРВЕРА>/value/<ТИП_МЕТРИКИ>/<ИМЯ_МЕТРИКИ>
	h.router.Get("/value/{type}/{name}", h.Get)
}

func (h *Handler) GetRouter() chi.Router {
	return h.router
}

func (h *Handler) Post(w http.ResponseWriter, r *http.Request) {
	metricType := chi.URLParam(r, "type")
	name := chi.URLParam(r, "name")
	value := chi.URLParam(r, "value")

	switch metricType {
	case "gauge":
		var gauge metrics.Gauge
		err := gauge.FromString(value)
		if err != nil {
			msg := fmt.Sprintf("value %v not acceptable - %v", name, err)
			http.Error(w, msg, http.StatusBadRequest)
			return
		}
		h.storage.Put(metrics.Name(name), gauge)
	case "counter":
		var counter metrics.Counter
		err := counter.FromString(value)
		if err != nil {
			msg := fmt.Sprintf("value %v not acceptable - %v", name, err)
			http.Error(w, msg, http.StatusBadRequest)
			return
		}
		h.storage.Count(metrics.Name(name), counter)
	default:
		err := fmt.Errorf("not implemented")
		http.Error(w, err.Error(), http.StatusNotImplemented)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	metricType := chi.URLParam(r, "type")
	name := chi.URLParam(r, "name")
	var val string

	switch metricType {
	case "gauge":
		gauge, err := h.storage.GetGauge(metrics.Name(name))
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		val = strconv.FormatFloat(float64(*gauge), 'f', -1, 64)
	case "counter":
		counter, err := h.storage.GetCounter(metrics.Name(name))
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		val = fmt.Sprintf("%d", *counter)
	default:
		err := fmt.Errorf("not implemented")
		http.Error(w, err.Error(), http.StatusNotImplemented)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(val))
}

func (h *Handler) List(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("content-type", "text/HTML")
	w.WriteHeader(http.StatusOK)

	var b bytes.Buffer
	b.WriteString("<h1>Current metrics data:</h1>")

	type gauge struct {
		key   string
		value float64
	}
	gauges := make([]gauge, 0, len(h.storage.Gauges))
	for k, val := range h.storage.Gauges {
		gauges = append(gauges, gauge{key: string(k), value: float64(val)})
	}
	sort.Slice(gauges, func(i, j int) bool { return gauges[i].key < gauges[j].key })

	b.WriteString(`<div><h2>Gauges</h2>`)
	for _, g := range gauges {
		val := strconv.FormatFloat(g.value, 'f', -1, 64)
		b.WriteString(fmt.Sprintf("<div>%s - %v</div>", g.key, val))
	}
	b.WriteString(`</div>`)

	b.WriteString(`<div><h2>Counters</h2>`)
	for k, val := range h.storage.Counters {
		b.WriteString(fmt.Sprintf("<div>%s - %d</div>", k, val))
	}
	b.WriteString(`</div>`)

	w.Write(b.Bytes())
}
