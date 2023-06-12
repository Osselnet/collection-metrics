package main

import (
	"github.com/Osselnet/metrics-collector/internal/server/handlers"
	"github.com/Osselnet/metrics-collector/internal/server/httpserver"
)

func main() {
	cfg := httpserver.Config{
		Address: "127.0.0.1",
		Port:    "8080",
	}
	h := handlers.New()
	httpserver.New(h, cfg)
}
