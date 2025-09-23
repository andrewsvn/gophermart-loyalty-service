package main

import (
	"log"
	"os"
	"os/signal"

	"github.com/andrewsvn/gophermart-ls/internal/app"
	"github.com/andrewsvn/gophermart-ls/internal/config"
	"github.com/andrewsvn/gophermart-ls/internal/logging"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg, err := config.GetServerConfig()
	if err != nil {
		return err
	}

	logger, err := logging.NewZapLogger(cfg.LogConfig)
	if err != nil {
		return err
	}

	gopherApp, err := app.NewGophermartApp(cfg, logger)
	if err != nil {
		return err
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	gopherApp.Start()
	<-stop
	gopherApp.Stop()

	return nil
}
