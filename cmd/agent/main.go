package main

import (
	"log"
	"time"
)

func main() {
	cfg := Config{
		Timeout:        4 * time.Second,
		PollInterval:   2 * time.Second,
		ReportInterval: 10 * time.Second,
		Address:        "127.0.0.1",
		Port:           ":8080",
	}
	agent, err := New(cfg)
	if err != nil {
		log.Fatal(err)
	}

	agent.Run()
}
