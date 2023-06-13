package main

import (
	"flag"
	"github.com/Osselnet/metrics-collector/internal/server/handlers"
	"github.com/Osselnet/metrics-collector/internal/server/httpserver"
	"os"
)

type Config struct {
	addr string
}

func main() {
	config := new(Config)
	flag.StringVar(&config.addr, "a", "127.0.0.1:8080", "address to listen on")
	flag.Parse()

	if a := os.Getenv("ADDRESS"); a != "" {
		config.addr = a
	}

	cfg := httpserver.Config{
		Addr: config.addr,
	}
	h := handlers.New()
	httpserver.New(h, cfg)
}
