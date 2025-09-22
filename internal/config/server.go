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
	AccrualIntegrationConfig

	URL         string `env:"RUN_ADDRESS" envDefault:"localhost:8080"`
	DatabaseURL string `env:"DATABASE_URI"`

	GracePeriodSec int `env:"GRACE_PERIOD" envDefault:"30"`
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
	cfg.AuthConfig.BindFlags()
	cfg.AccrualIntegrationConfig.BindFlags()

	flag.StringVar(&cfg.URL, "a", cfg.URL,
		"The address to run the service on")
	flag.StringVar(&cfg.DatabaseURL, "d", cfg.DatabaseURL,
		"The address of the postgres database")
}

func (cfg *ServerConfig) Validate() error {
	if cfg.URL == "" {
		return fmt.Errorf("url is required")
	}
	if cfg.DatabaseURL == "" {
		return fmt.Errorf("databaseUrl is required")
	}
	return nil
}
