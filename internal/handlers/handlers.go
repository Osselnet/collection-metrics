package handlers

import (
	"fmt"
	"github.com/Osselnet/metrics-collector/internal/memstorage"
	"github.com/Osselnet/metrics-collector/pkg/metrics"
	"net/http"
	"strings"
)

type Gauge struct {
	*memstorage.MemStorage
}

type Counter struct {
	*memstorage.MemStorage
}

func (g *Gauge) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	params := strings.Split(r.URL.Path, "/")

	if len(params) != 5 {
		http.Error(w, "Wrong parameters number!", http.StatusBadRequest)
		return
	}
	if params[4] == "" {
		http.Error(w, "Wrong value!", http.StatusNotAcceptable)
		return
	}

	metricName := metrics.Name(params[3])

	var gauge metrics.Gauge
	err := gauge.FromString(params[4])
	if err != nil {
		msg := fmt.Sprintf("value %v not acceptable - %v", params[4], err)
		http.Error(w, msg, http.StatusNotAcceptable)
		return
	}
	g.Put(metricName, gauge)

	w.WriteHeader(http.StatusOK)
}

func (c *Counter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	params := strings.Split(r.URL.Path, "/")

	if len(params) != 5 {
		http.Error(w, "Wrong parameters number!", http.StatusBadRequest)
		return
	}
	if params[4] == "" {
		http.Error(w, "Wrong value!", http.StatusBadRequest)
		return
	}

	metricName := metrics.Name(params[3])

	var counter metrics.Counter
	err := counter.FromString(params[4])
	if err != nil {
		msg := fmt.Sprintf("value %v not acceptable - %v", params[4], err)
		http.Error(w, msg, http.StatusNotAcceptable)
		return
	}
	c.Count(metricName, counter)

	w.WriteHeader(http.StatusOK)
}
