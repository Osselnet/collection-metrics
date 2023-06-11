package main

import (
	"github.com/Osselnet/metrics-collector/internal/handlers"
	"github.com/Osselnet/metrics-collector/internal/storage"
	"github.com/Osselnet/metrics-collector/pkg/metrics"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestAgent_sendReport(t *testing.T) {
	type want struct {
		statusCode int
	}
	type request struct {
		key   metrics.Name
		value any
	}

	tests := []struct {
		name string
		req  request
		want want
	}{
		{
			name: "Test Valid Post request gauge metric",
			req: request{
				key:   "HeapObjects",
				value: metrics.Gauge(4242.23),
			},
			want: want{
				statusCode: http.StatusOK,
			},
		},
		{
			name: "Test Valid Post request counter metric",
			req: request{
				key:   "PollCount",
				value: metrics.Counter(100),
			},
			want: want{
				statusCode: http.StatusOK,
			},
		},
		{
			name: "Test Invalid Post request counter metric",
			req: request{
				key:   "MCacheSys",
				value: "ab",
			},
			want: want{
				statusCode: http.StatusBadRequest,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st := storage.New()
			gaugeHandler := &handlers.Gauge{MemStorage: st}
			counterHandler := &handlers.Counter{MemStorage: st}

			mux := http.NewServeMux()
			mux.Handle("/update/gauge/", gaugeHandler)
			mux.Handle("/update/counter/", counterHandler)

			server := httptest.NewServer(http.Handler(mux))
			defer server.Close()

			params := strings.Split(server.URL, ":")
			cfg := Config{
				Timeout:        4 * time.Second,
				PollInterval:   2 * time.Second,
				ReportInterval: 10 * time.Second,
				Address:        "127.0.0.1",
				Port:           ":" + params[len(params)-1],
			}
			a, err := New(cfg)
			assert.NoError(t, err)

			statusCode := a.sendRequest(tt.req.key, tt.req.value)
			assert.Equal(t, tt.want.statusCode, statusCode)
		})
	}
}
