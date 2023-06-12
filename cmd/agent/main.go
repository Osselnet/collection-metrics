package main

import (
	agent "github.com/Osselnet/metrics-collector/internal/agent"
	"log"
	"time"
)

func main() {
	cfg := agent.Config{
		Timeout:        4 * time.Second,
		PollInterval:   2 * time.Second,
		ReportInterval: 10 * time.Second,
		Address:        "127.0.0.1",
		Port:           ":8080",
	}
	agent, err := agent.New(cfg)
	if err != nil {
		log.Fatal(err)
	}

	agent.Run()
}
