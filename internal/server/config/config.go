package config

import (
	"flag"
	"fmt"
	"github.com/caarlos0/env"
)

type Config struct {
	Address  string `env:"ADDRESS"`
	Interval int    `env:"STORE_INTERVAL"`
	Filename string `env:"FILE_STORAGE_PATH"`
	Restore  bool   `env:"RESTORE"`
	DSN      string `env:"DATABASE_DSN"`
}

func ParseConfig() (Config, error) {
	var cfg Config

	flag.StringVar(&cfg.Address,
		"a", "localhost:8080",
		"Add addres and port in format <address>:<port>")
	flag.IntVar(&cfg.Interval,
		"i", 300,
		"Saving metrics to file interval")
	flag.StringVar(&cfg.Filename,
		"f", "/tmp/metrics-db.json",
		"File path")
	flag.BoolVar(&cfg.Restore,
		"r", true,
		"Restore metrics value from file")
	flag.StringVar(&cfg.DSN,
		"d", fmt.Sprintf(
			"host=%s port=%d dbname=%s user=%s password=%s target_session_attrs=read-write",
			"127.0.0.1", 5432, "postgres", "pass", "postgres"),
		"Connection string in Postgres format")

	flag.Parse()

	err := env.Parse(&cfg)
	if err != nil {
		return cfg, err
	}

	return cfg, nil
}
