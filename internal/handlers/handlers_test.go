package handlers

import (
	"github.com/Osselnet/metrics-collector/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestServeHTTP(t *testing.T) {
	type want struct {
		statusCode int
	}
	tests := []struct {
		name    string
		request string
		handler interface{}
		want    want
	}{
		{
			name:    "ok gauge",
			request: "/update/gauge/Alloc/1234.567",
			handler: Gauge{},
			want: want{
				statusCode: http.StatusOK,
			},
		},
		{
			name:    "ok counter",
			request: "/update/counter/PollCount/1",
			handler: Counter{},
			want: want{
				statusCode: http.StatusOK,
			},
		},
		{
			name:    "gauge 400",
			handler: Gauge{},
			request: "/update/gauge/Alloc/OtherSys",
			want: want{
				statusCode: http.StatusBadRequest,
			},
		},
		{
			name:    "counter 400",
			handler: Counter{},
			request: "/update/counter/PollCount/RandomValue",
			want: want{
				statusCode: http.StatusBadRequest,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, tt.request, nil)
			w := httptest.NewRecorder()

			switch tt.handler.(type) {
			case Gauge:
				h := tt.handler.(Gauge)
				h.MemStorage = storage.New()
				h.ServeHTTP(w, request)
			case Counter:
				h := tt.handler.(Counter)
				h.MemStorage = storage.New()
				h.ServeHTTP(w, request)
			}

			result := w.Result()

			assert.Equal(t, tt.want.statusCode, result.StatusCode)

			_, err := ioutil.ReadAll(result.Body)
			require.NoError(t, err)
			err = result.Body.Close()
			require.NoError(t, err)
		})
	}
}
