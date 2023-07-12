package agent

import (
	"github.com/Osselnet/metrics-collector/internal/server/handlers"
	"github.com/Osselnet/metrics-collector/internal/storage"
	"github.com/Osselnet/metrics-collector/pkg/metrics"
	"github.com/go-chi/chi/v5"
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
			name: "Test Invalid Post request",
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
			h := handlers.New(chi.NewRouter(), &storage.MemStorage{
				Metrics: metrics.New(),
			}, nil, "", false)
			server := httptest.NewServer(h.GetRouter())
			defer server.Close()

			params := strings.Split(server.URL, ":")
			cfg := Config{
				Timeout:        4 * time.Second,
				PollInterval:   2 * time.Second,
				ReportInterval: 10 * time.Second,
				Address:        "127.0.0.1:" + params[len(params)-1],
			}
			a, err := New(cfg)
			assert.NoError(t, err)

			statusCode := a.sendRequest(tt.req.key, tt.req.value)
			assert.Equal(t, tt.want.statusCode, statusCode)
		})
	}
}
