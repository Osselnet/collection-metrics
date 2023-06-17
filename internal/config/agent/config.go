package agent

import (
	"flag"
	"github.com/caarlos0/env"
	"os"
)

type Config struct {
	Addr           string `env:"ADDRESS" envDefault:"127.0.0.1:8080"`
	ReportInterval int    `env:"REPORT_INTERVAL" envDefault:"10"`
	PollInterval   int    `env:"POLL_INTERVAL" envDefault:"2"`
}

func ParseConfig() (Config, error) {
	config := new(Config)
	flag.StringVar(&config.Addr, "a", "127.0.0.1:8080", "address to listen on")
	flag.IntVar(&config.ReportInterval, "r", 10, "write metrics to file interval")
	flag.IntVar(&config.PollInterval, "p", 2, "write metrics to file interval")
	flag.Parse()

	envConfig := Config{}
	err := env.Parse(&envConfig)
	if err != nil {
		return *config, err
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

	return *config, nil
}
