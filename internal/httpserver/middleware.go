package httpserver

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

func panicMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				msg := fmt.Sprintf("recover server after panic - %s", err)

				log.Println(msg)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintf(w, "{\"Error\": \"%s\"}", msg)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func accessLogMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("[%s] %s, %s, %s | %s\n",
			r.Method, r.RemoteAddr, r.URL.Path, r.UserAgent(), time.Since(start))
	})
}

func preChecksMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Only POST requests are allowed!", http.StatusBadRequest)
			return
		}

		params := strings.Split(r.URL.Path, "/")

		if len(params) != 5 {
			http.Error(w, "Wrong parameters number!", http.StatusNotFound)
			return
		}
		if params[4] == "" {
			http.Error(w, "Wrong value!", http.StatusNotFound)
			return
		}
		if params[2] != "gauge" && params[2] != "counter" {
			http.Error(w, "Incorrect metric type!", http.StatusBadRequest)
			return
		}

		next.ServeHTTP(w, r)
	})
}
