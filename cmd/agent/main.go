package main

import (
	"flag"
	"github.com/Osselnet/metrics-collector/internal/agent"
	"github.com/caarlos0/env"
	"log"
	"os"
	"time"
)

type Config struct {
	Addr           string `env:"ADDRESS" envDefault:"127.0.0.1:8080"`
	ReportInterval int    `env:"REPORT_INTERVAL" envDefault:"10"`
	PollInterval   int    `env:"POLL_INTERVAL" envDefault:"2"`
}

func main() {
	config := new(Config)
	flag.StringVar(&config.Addr, "a", "127.0.0.1:8080", "address to listen on")
	flag.IntVar(&config.ReportInterval, "r", 10, "write metrics to file interval")
	flag.IntVar(&config.PollInterval, "p", 2, "write metrics to file interval")
	flag.Parse()

	envConfig := Config{}
	err := env.Parse(&envConfig)
	if err != nil {
		log.Println("Error -", err)
	}

	if _, ok := os.LookupEnv("ADDRESS"); ok {
		config.Addr = envConfig.Addr
	}
	if _, ok := os.LookupEnv("REPORT_INTERVAL"); ok {
		config.ReportInterval = envConfig.ReportInterval
	}
	if _, ok := os.LookupEnv("POLL_INTERVAL"); ok {
		config.PollInterval = envConfig.PollInterval
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
