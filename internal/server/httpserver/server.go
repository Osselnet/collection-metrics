package httpserver

import (
	"github.com/Osselnet/metrics-collector/internal/server/handlers"
	"net/http"
)

type Config struct {
	Address string
	Port    string
}

func New(h *handlers.Handler, cfg Config) {

	addr := cfg.Address + ":" + cfg.Port
	err := http.ListenAndServe(addr, h.GetRouter())
	if err != nil {
		panic(err)
	}
}
