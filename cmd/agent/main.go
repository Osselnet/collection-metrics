package main

import (
	"flag"
	agent "github.com/Osselnet/metrics-collector/internal/agent"
	"log"
	"time"
)

type Config struct {
	addr           string
	reportInterval time.Duration
	pollInterval   time.Duration
}

func main() {
	config := new(Config)
	flag.StringVar(&config.addr, "a", "127.0.0.1:8080", "address to listen on")
	flag.DurationVar(&config.reportInterval, "r", 10*time.Second, "write metrics to file interval")
	flag.DurationVar(&config.pollInterval, "p", 2*time.Second, "write metrics to file interval")
	flag.Parse()

	cfg := agent.Config{
		Timeout:        4 * time.Second,
		PollInterval:   config.pollInterval,
		ReportInterval: config.reportInterval,
		Address:        config.addr,
	}
	agent, err := agent.New(cfg)
	if err != nil {
		log.Fatal(err)
	}

	agent.Run()
}
