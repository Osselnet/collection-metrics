package config

import (
	"flag"
	"github.com/caarlos0/env"
	"os"
)

type Config struct {
	Address  string `env:"ADDRESS"`
	Interval int    `env:"STORE_INTERVAL"`
	Filename string `env:"FILE_STORAGE_PATH"`
	Restore  bool   `env:"RESTORE"`
	DSN      string `env:"DATABASE_DSN"`
	Key      string `env:"KEY"`
}

func ParseConfig() (Config, error) {
	config := new(Config)

	flag.StringVar(&config.Address,
		"a", "localhost:8080",
		"Add addres and port in format <address>:<port>")
	flag.IntVar(&config.Interval,
		"i", 300,
		"Saving metrics to file interval")
	flag.StringVar(&config.Filename,
		"f", "/tmp/metrics-db.json",
		"File path")
	flag.BoolVar(&config.Restore,
		"r", true,
		"Restore metrics value from file")
	flag.StringVar(&config.DSN,
		"d", "",
		"Connection string in Postgres format")
	flag.StringVar(&config.Key,
		"k", "",
		"Sing key")

	flag.Parse()

	envConfig := Config{}
	err := env.Parse(&envConfig)
	if err != nil {
		return *config, err
	}

	if _, ok := os.LookupEnv("ADDRESS"); ok {
		config.Address = envConfig.Address
	}
	if _, ok := os.LookupEnv("STORE_INTERVAL"); ok {
		config.Interval = envConfig.Interval
	}
	if _, ok := os.LookupEnv("FILE_STORAGE_PATH"); ok {
		config.Filename = envConfig.Filename
	}
	if _, ok := os.LookupEnv("RESTORE"); ok {
		config.Restore = envConfig.Restore
	}
	if _, ok := os.LookupEnv("DATABASE_DSN"); ok {
		config.DSN = envConfig.DSN
	}
	if _, ok := os.LookupEnv("KEY"); ok {
		config.Key = envConfig.Key
	}

	return *config, nil
}
