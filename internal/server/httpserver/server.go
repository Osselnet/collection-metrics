package httpserver

import (
	"github.com/Osselnet/metrics-collector/internal/server/handlers"
	"net/http"
)

type Config struct {
	Addr string
}

func New(h *handlers.Handler, cfg Config) {
	err := http.ListenAndServe(cfg.Addr, h.GetRouter())
	if err != nil {
		panic(err)
	}
}
