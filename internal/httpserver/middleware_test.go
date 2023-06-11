package httpserver

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

func Test_preChecksMiddleware(t *testing.T) {
	type want struct {
		statusCode int
	}

	tests := []struct {
		name        string
		method      string
		target      string
		contentType string
		want        want
	}{
		{
			name:        "Method not allowed",
			method:      http.MethodGet,
			target:      "/update/gauge/Alloc/1234.567",
			contentType: "text/plain",
			want: want{
				statusCode: http.StatusBadRequest,
			},
		},
		{
			name:        "counter 404",
			method:      http.MethodPost,
			target:      "/update/counter/",
			contentType: "text/plain",
			want: want{
				statusCode: http.StatusNotFound,
			},
		},
		{
			name:        "gauge 404",
			method:      http.MethodPost,
			target:      "/update/gauge/",
			contentType: "text/plain",
			want: want{
				statusCode: http.StatusNotFound,
			},
		},
		{
			name:        "Status ok",
			method:      http.MethodPost,
			target:      "/update/counter/testCounter/42",
			contentType: "text/plain",
			want: want{
				statusCode: http.StatusOK,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(tt.method, tt.target, nil)
			request.Header.Set("Content-Type", tt.contentType)
			w := httptest.NewRecorder()

			mux := http.NewServeMux()
			mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})
			preChecksMiddleware(mux).ServeHTTP(w, request)

			result := w.Result()

			assert.Equal(t, tt.want.statusCode, result.StatusCode)

			_, err := ioutil.ReadAll(result.Body)
			require.NoError(t, err)
			err = result.Body.Close()
			require.NoError(t, err)
		})
	}
}
