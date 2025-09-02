package main

import (
	"log"

	"github.com/andrewsvn/gophermart-ls/internal/config"
	"github.com/andrewsvn/gophermart-ls/internal/db"
	"github.com/andrewsvn/gophermart-ls/internal/logging"
	"go.uber.org/zap"
)

func main() {
	cfg, err := config.GetServerConfig()
	if err != nil {
		log.Fatal(err)
	}

	logger, err := logging.NewZapLogger(cfg.LogConfig)
	if err != nil {
		log.Fatal(err)
	}

	logger.Info("starting gophermart-loyalty-service",
		zap.String("service URL", cfg.Url),
		zap.String("postgres DB URL", cfg.DatabaseUrl),
		zap.String("loyalty accrual service URL", cfg.AccrualServiceUrl),
	)

	logger.Info("migrating database schema")
	err = db.Migrate(cfg.DatabaseUrl, logger)
	if err != nil {
		log.Fatal(err)
	}

}
