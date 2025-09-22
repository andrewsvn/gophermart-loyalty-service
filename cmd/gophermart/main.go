package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"

	"github.com/andrewsvn/gophermart-ls/internal/auth"
	"github.com/andrewsvn/gophermart-ls/internal/config"
	"github.com/andrewsvn/gophermart-ls/internal/db"
	"github.com/andrewsvn/gophermart-ls/internal/handlers"
	"github.com/andrewsvn/gophermart-ls/internal/integration/accrual"
	"github.com/andrewsvn/gophermart-ls/internal/logging"
	"github.com/andrewsvn/gophermart-ls/internal/repository/postgres"
	"github.com/andrewsvn/gophermart-ls/internal/service"
	"go.uber.org/zap"
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

	logger.Info("starting gophermart-loyalty-service",
		zap.String("service URL", cfg.URL),
		zap.String("loyalty accrual service URL", cfg.AccrualServiceURL),
	)

	logger.Info("migrating database schema")
	err = db.Migrate(cfg.DatabaseURL, logger)
	if err != nil {
		return err
	}

	ctx, done := context.WithCancel(context.Background())
	defer done()

	logger.Info("initializing storage")
	pgdb, err := db.NewPostgresDB(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pgdb.Close()

	storageManager := postgres.NewPgStorageManager(pgdb)

	logger.Info("initializing identity provider")
	idp, err := auth.NewIdentityProvider(&cfg.AuthConfig, storageManager.GetUserStorage(), logger)
	if err != nil {
		return err
	}

	logger.Info("initializing service layer")
	userService := service.NewUserService(storageManager.GetUserStorage(), idp, logger)
	orderService := service.NewOrderService(storageManager.GetLoyaltyStorage(), logger)

	logger.Info("initializing HTTP server")
	userHandlers := handlers.NewUserManagementHandlers(userService)
	orderHandlers := handlers.NewOrderManagementHandlers(orderService, idp, logger)
	server := handlers.NewRestServer(cfg, logger,
		userHandlers,
		orderHandlers,
	)

	logger.Info("initializing accrual system integration")
	accrualInt := accrual.NewIntegrationFlow(&cfg.AccrualIntegrationConfig, storageManager.GetLoyaltyStorage(), logger)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	logger.Info("starting application routines")
	server.Start()
	accrualInt.Start()

	<-stop
	logger.Info("shutting down application routines")

	wg := &sync.WaitGroup{}
	wg.Add(2)
	go func() {
		defer wg.Done()
		defer logging.Sync(logger)
		server.GracefulShutdown()
	}()
	go func() {
		defer wg.Done()
		defer logging.Sync(logger)
		accrualInt.Shutdown()
	}()
	wg.Wait()
	return nil
}
