package main

import "github.com/Osselnet/metrics-collector/internal/httpserver"

func main() {
	cfg := httpserver.Config{
		Address: "127.0.0.1",
		Port:    "8080",
	}
	httpserver.New(cfg)
}
