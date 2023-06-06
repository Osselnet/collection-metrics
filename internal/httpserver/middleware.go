package httpserver

import (
	"fmt"
	"log"
	"net/http"
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
		if r.Method != "POST" {
			http.Error(w, "Only POST requests are allowed!", http.StatusBadRequest)
			return
		}

		if r.Header.Get("Content-Type") != "text/plain" {
			http.Error(w, "Only text/plain content-type allowed!", http.StatusBadRequest)
			return
		}

		next.ServeHTTP(w, r)
	})
}
