package config

import "flag"

type LogConfig struct {
	Level string `env:"LOG_LEVEL" envDefault:"info"`
}

func (cfg *LogConfig) BindFlags() {
	flag.StringVar(&cfg.Level, "lvl", cfg.Level, "Application log level")
}
