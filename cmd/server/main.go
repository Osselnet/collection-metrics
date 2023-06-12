package main

import (
	"flag"
	"github.com/Osselnet/metrics-collector/internal/server/handlers"
	"github.com/Osselnet/metrics-collector/internal/server/httpserver"
)

type Config struct {
	addr string
}

func main() {
	config := new(Config)
	flag.StringVar(&config.addr, "a", "127.0.0.1:8080", "address to listen on")
	flag.Parse()

	cfg := httpserver.Config{
		Addr: config.addr,
	}
	h := handlers.New()
	httpserver.New(h, cfg)
}
