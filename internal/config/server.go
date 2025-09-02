package config

import (
	"errors"
	"flag"
	"fmt"

	"github.com/caarlos0/env/v6"
)

var (
	ErrParseConfig = errors.New("error parsing config")
)

type ServerConfig struct {
	LogConfig
	AuthConfig

	Url               string `env:"RUN_ADDRESS" envDefault:"http://localhost:8080"`
	AccrualServiceUrl string `env:"ACCRUAL_SYSTEM_ADDRESS"`
	DatabaseUrl       string `env:"DATABASE_URI" envDefault:"localhost:5432"`
}

func GetServerConfig() (*ServerConfig, error) {
	cfg := ServerConfig{}
	err := env.Parse(&cfg)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrParseConfig, err)
	}
	cfg.BindFlags()
	flag.Parse()

	err = cfg.Validate()
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrParseConfig, err)
	}

	return &cfg, nil
}

func (cfg *ServerConfig) BindFlags() {
	cfg.LogConfig.BindFlags()

	flag.StringVar(&cfg.Url, "a", cfg.Url,
		"The address to run the service on")
	flag.StringVar(&cfg.AccrualServiceUrl, "r", cfg.AccrualServiceUrl,
		"The address of the accrual service")
	flag.StringVar(&cfg.DatabaseUrl, "d", cfg.DatabaseUrl,
		"The address of the postgres database")
}

func (cfg *ServerConfig) Validate() error {
	if cfg.Url == "" {
		return fmt.Errorf("url is required")
	}
	if cfg.AccrualServiceUrl == "" {
		return fmt.Errorf("accrual service url is required")
	}
	if cfg.DatabaseUrl == "" {
		return fmt.Errorf("databaseUrl is required")
	}
	return nil
}
