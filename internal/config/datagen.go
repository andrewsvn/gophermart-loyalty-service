package config

import (
	"flag"
	"fmt"

	"github.com/caarlos0/env/v6"
)

type DatagenConfig struct {
	LogConfig

	DatabaseURL string `env:"DATABASE_URI"`
	IsCleanup   bool
}

func GetDatagenConfig() (*DatagenConfig, error) {
	cfg := DatagenConfig{}

	err := env.Parse(&cfg)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrParseConfig, err)
	}

	cfg.LogConfig.BindFlags()
	flag.StringVar(&cfg.DatabaseURL, "d", cfg.DatabaseURL,
		"The address of the postgres database")
	flag.BoolVar(&cfg.IsCleanup, "clean", false, "cleanup mode to erase all test data")
	flag.Parse()

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URI not provided")
	}
	return &cfg, nil
}
