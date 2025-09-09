package config

import (
	"flag"
	"fmt"

	"github.com/caarlos0/env/v6"
)

type DatagenConfig struct {
	LogConfig

	DatabaseUrl string `env:"DATABASE_URI"`
}

func GetDatagenConfig() (*DatagenConfig, error) {
	cfg := DatagenConfig{}

	err := env.Parse(&cfg)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrParseConfig, err)
	}

	cfg.LogConfig.BindFlags()
	flag.StringVar(&cfg.DatabaseUrl, "d", cfg.DatabaseUrl,
		"The address of the postgres database")
	flag.Parse()

	if cfg.DatabaseUrl == "" {
		return nil, fmt.Errorf("DATABASE_URI not provided")
	}
	return &cfg, nil
}
