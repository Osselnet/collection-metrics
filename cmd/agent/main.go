package main

import (
	"github.com/Osselnet/metrics-collector/internal/agent"
	config "github.com/Osselnet/metrics-collector/internal/config/agent"
	"log"
	"time"
)

func main() {
	config, err := config.ParseConfig()
	if err != nil {
		log.Println("Error -", err)
	}

	cfg := agent.Config{
		Timeout:        4 * time.Second,
		PollInterval:   time.Duration(config.PollInterval) * time.Second,
		ReportInterval: time.Duration(config.ReportInterval) * time.Second,
		Address:        config.Addr,
	}

	agent, err := agent.New(cfg)
	if err != nil {
		log.Fatal(err)
	}

	agent.Run()
}
