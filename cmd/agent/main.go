package main

import (
	"flag"
	agent "github.com/Osselnet/metrics-collector/internal/agent"
	"log"
	"time"
)

type Config struct {
	addr           string
	reportInterval int
	pollInterval   int
}

func main() {
	config := new(Config)
	flag.StringVar(&config.addr, "a", "127.0.0.1:8080", "address to listen on")
	flag.IntVar(&config.reportInterval, "r", 10, "write metrics to file interval")
	flag.IntVar(&config.pollInterval, "p", 2, "write metrics to file interval")
	flag.Parse()

	cfg := agent.Config{
		Timeout:        4 * time.Second,
		PollInterval:   time.Duration(config.pollInterval) * time.Second,
		ReportInterval: time.Duration(config.reportInterval) * time.Second,
		Address:        config.addr,
	}
	agent, err := agent.New(cfg)
	if err != nil {
		log.Fatal(err)
	}

	agent.Run()
}
