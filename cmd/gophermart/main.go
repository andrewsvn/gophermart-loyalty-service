package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"github.com/andrewsvn/gophermart-ls/internal/auth"
	"github.com/andrewsvn/gophermart-ls/internal/config"
	"github.com/andrewsvn/gophermart-ls/internal/db"
	"github.com/andrewsvn/gophermart-ls/internal/handlers"
	"github.com/andrewsvn/gophermart-ls/internal/logging"
	"github.com/andrewsvn/gophermart-ls/internal/repository"
	"github.com/andrewsvn/gophermart-ls/internal/service"
	"go.uber.org/zap"
)

func main() {
	log.Fatal(run())
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
		zap.String("service URL", cfg.Url),
		zap.String("postgres DB URL", cfg.DatabaseUrl),
		zap.String("loyalty accrual service URL", cfg.AccrualServiceUrl),
	)

	logger.Info("migrating database schema")
	err = db.Migrate(cfg.DatabaseUrl, logger)
	if err != nil {
		return err
	}

	ctx, done := context.WithCancel(context.Background())
	defer done()

	logger.Info("initializing storage")
	pgdb, err := db.NewPostgresDB(ctx, cfg.DatabaseUrl)
	if err != nil {
		return err
	}
	defer pgdb.Close()

	userRepo := repository.NewUserRepository(pgdb)
	orderRepo := repository.NewOrderRepository(pgdb)
	withdrawalRepo := repository.NewWithdrawalRepository(pgdb)

	logger.Info("initializing identity provider")
	idp, err := auth.NewIdentityProvider(&cfg.AuthConfig, userRepo)
	if err != nil {
		return err
	}

	logger.Info("initializing service layer")
	userService := service.NewUserService(userRepo, idp, logger)
	orderService := service.NewOrderService(orderRepo, withdrawalRepo, logger)

	logger.Info("initializing HTTP server")
	userHandlers := handlers.NewUserManagementHandlers(userService)
	orderHandlers := handlers.NewOrderManagementHandlers(orderService, idp, logger)
	server := handlers.NewRestServer(cfg, logger,
		userHandlers,
		orderHandlers,
	)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	server.Start()

	<-stop
	logger.Info("shutting down HTTP server")
	server.GracefulShutdown()
	return nil
}
