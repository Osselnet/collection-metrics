package handlers

import (
	"bytes"
	"fmt"
	"github.com/Osselnet/metrics-collector/pkg/metrics"
	"github.com/go-chi/chi/v5"
	"net/http"
	"sort"
	"strconv"
)

func (h *Handler) Post() http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
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
	return http.HandlerFunc(fn)
}

func (h *Handler) Get() http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
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
	return http.HandlerFunc(fn)
}

func (h *Handler) List() http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
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
	return http.HandlerFunc(fn)
}
