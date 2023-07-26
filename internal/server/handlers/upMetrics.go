package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/Osselnet/metrics-collector/pkg/metrics"
	"github.com/go-chi/chi/v5"
	"log"
	"net/http"
	"sort"
	"strconv"
)

type Metrics struct {
	ID    string   `json:"id"`              // имя метрики
	MType string   `json:"type"`            // параметр, принимающий значение gauge или counter
	Delta *int64   `json:"delta,omitempty"` // значение метрики в случае передачи counter
	Value *float64 `json:"value,omitempty"` // значение метрики в случае передачи gauge
	Hash  string   `json:"hash,omitempty"`  // значение хеш-функции
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
		h.Storage.Put(r.Context(), name, gauge)
	case "counter":
		var counter metrics.Counter
		err := counter.FromString(value)
		if err != nil {
			msg := fmt.Sprintf("value %v not acceptable - %v", name, err)
			http.Error(w, msg, http.StatusBadRequest)
			return
		}
		h.Storage.Put(r.Context(), name, counter)
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
		gauge, err := h.Storage.Get(r.Context(), name)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		val = strconv.FormatFloat(float64(gauge.(metrics.Gauge)), 'f', -1, 64)
	case "counter":
		counter, err := h.Storage.Get(r.Context(), name)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		val = fmt.Sprintf("%d", counter.(metrics.Counter))
	default:
		err := fmt.Errorf("not implemented")
		http.Error(w, err.Error(), http.StatusNotImplemented)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(val))
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	var b bytes.Buffer
	b.WriteString("<h1>Current metrics data:</h1>")

	type gauge struct {
		key   string
		value float64
	}

	mcs, err := h.Storage.GetMetrics(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	gauges := make([]gauge, 0, metrics.GaugeLen)
	for k, val := range mcs.Gauges {
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
	for k, val := range mcs.Counters {
		b.WriteString(fmt.Sprintf("<div>%s - %d</div>", k, val))
	}
	b.WriteString(`</div>`)

	w.Write(b.Bytes())
}

func (h *Handler) JSONValue(w http.ResponseWriter, r *http.Request) {
	var m Metrics

	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&m)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	switch m.MType {
	case "counter":
		counter, err := h.Storage.Get(r.Context(), m.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		v := int64(counter.(metrics.Counter))
		m.Delta = &v
		if h.key != "" {
			m.Hash = metrics.CounterHash(h.key, m.ID, *m.Delta)
		}
	case "gauge":
		gauge, err := h.Storage.Get(r.Context(), m.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		v := float64(gauge.(metrics.Gauge))
		m.Value = &v
		if h.key != "" {
			m.Hash = metrics.GaugeHash(h.key, m.ID, *m.Value)
		}
	}

	resp, err := json.Marshal(m)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}

func (h *Handler) JSONUpdate(w http.ResponseWriter, r *http.Request) {
	var m Metrics
	var buf bytes.Buffer

	_, err := buf.ReadFrom(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = json.Unmarshal(buf.Bytes(), &m)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	switch m.MType {
	case metrics.TypeCounter:
		if m.Delta == nil {
			http.Error(w, "metric value should not be empty", http.StatusBadRequest)
			return
		}
		if h.key != "" && m.Hash != "" {
			if metrics.CounterHash(h.key, m.ID, *m.Delta) != m.Hash {
				err = fmt.Errorf("hash check failed for counter metric")
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		}
		h.Storage.Put(r.Context(), m.ID, metrics.Counter(*m.Delta))
		w.WriteHeader(http.StatusOK)
	case metrics.TypeGauge:
		if m.Value == nil {
			http.Error(w, "metric value should not be empty", http.StatusBadRequest)
			return
		}
		if h.key != "" && m.Hash != "" {
			if metrics.GaugeHash(h.key, m.ID, *m.Value) != m.Hash {
				err = fmt.Errorf("hash check failed for gauge metric")
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		}
		h.Storage.Put(r.Context(), m.ID, metrics.Gauge(*m.Value))
		w.WriteHeader(http.StatusOK)
	default:
		http.Error(w, "Incorrect metric type", http.StatusBadRequest)
	}
}

func (h *Handler) Ping(w http.ResponseWriter, r *http.Request) {
	if h.dbStorage == nil {
		http.Error(w, "database not plugged in", http.StatusInternalServerError)
		return
	}

	err := h.dbStorage.Ping(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) HandleBatchUpdate(w http.ResponseWriter, r *http.Request) {
	var m []Metrics
	var buf bytes.Buffer

	_, err := buf.ReadFrom(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = json.Unmarshal(buf.Bytes(), &m)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	for _, v := range m {
		switch v.MType {
		case metrics.TypeCounter:
			if v.Delta == nil {
				http.Error(w, "metric value should not be empty", http.StatusBadRequest)
				return
			}
			h.Storage.Put(r.Context(), v.ID, metrics.Counter(*v.Delta))
		case metrics.TypeGauge:
			if v.Value == nil {
				http.Error(w, "metric value should not be empty", http.StatusBadRequest)
				return
			}
			h.Storage.Put(r.Context(), v.ID, metrics.Gauge(*v.Value))
		default:
			http.Error(w, "Incorrect metric type", http.StatusBadRequest)
		}
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) hashCheck(m *Metrics) error {
	if h.key == "" {
		return nil
	}
	switch m.MType {
	case "gauge":
		if m.Value == nil {
			m.Value = new(float64)
		}

		mac1 := m.Hash
		mac2 := metrics.GaugeHash(h.key, m.ID, *m.Value)
		if mac1 != mac2 {
			log.Printf(":: mac1 - %s\n", mac1)
			log.Printf(":: mac2 - %s\n", mac2)
			return fmt.Errorf("hash check failed for gauge metric")
		}
	case "counter":
		if m.Delta == nil {
			m.Delta = new(int64)
		}

		mac1 := m.Hash
		mac2 := metrics.CounterHash(h.key, m.ID, *m.Delta)
		if mac1 != mac2 {
			return fmt.Errorf("hash check failed for counter metric")
		}
	default:
		err := fmt.Errorf("not implemented")
		return err
	}
	return nil
}
