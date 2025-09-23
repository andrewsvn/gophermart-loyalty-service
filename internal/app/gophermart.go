package app

import (
	"sync"

	"github.com/andrewsvn/gophermart-ls/internal/auth"
	"github.com/andrewsvn/gophermart-ls/internal/config"
	"github.com/andrewsvn/gophermart-ls/internal/db"
	"github.com/andrewsvn/gophermart-ls/internal/handlers"
	"github.com/andrewsvn/gophermart-ls/internal/integration/accrual"
	"github.com/andrewsvn/gophermart-ls/internal/logging"
	"github.com/andrewsvn/gophermart-ls/internal/repository"
	"github.com/andrewsvn/gophermart-ls/internal/repository/postgres"
	"github.com/andrewsvn/gophermart-ls/internal/service"
	"go.uber.org/zap"
)

type GophermartApp struct {
	cfg    *config.ServerConfig
	logger *zap.Logger

	stor           repository.Manager
	idp            *auth.IdentityProvider
	userService    *service.UserService
	loyaltyService *service.LoyaltyService
	restServer     *handlers.RestServer
	accrualIntFlow *accrual.IntegrationFlow
}

func NewGophermartApp(cfg *config.ServerConfig, l *zap.Logger) (*GophermartApp, error) {
	sl := &GophermartApp{
		cfg:    cfg,
		logger: l,
	}

	sl.logger.Info("initializing gophermart-loyalty-service",
		zap.String("service URL", sl.cfg.URL),
		zap.String("loyalty accrual service URL", sl.cfg.AccrualServiceURL),
	)

	err := sl.initStorage()
	if err != nil {
		return nil, err
	}

	sl.initIdentityProvider()
	sl.initRestServer()
	sl.initAccrualIntegrationFlow()

	return sl, nil
}

func (sl *GophermartApp) Start() {
	sl.logger.Info("starting gophermart-loyalty-service")
	sl.restServer.Start()
	sl.accrualIntFlow.Start()
}

func (sl *GophermartApp) Stop() {
	wg := &sync.WaitGroup{}
	wg.Add(2)
	go func() {
		defer wg.Done()
		defer logging.Sync(sl.logger)
		sl.restServer.GracefulShutdown()
	}()
	go func() {
		defer wg.Done()
		defer logging.Sync(sl.logger)
		sl.accrualIntFlow.Shutdown()
	}()
	wg.Wait()

	sl.stor.Close()
	sl.logger.Info("gophermart-loyalty-service stopped")
}

func (sl *GophermartApp) initStorage() error {
	sl.logger.Info("migrating database schema")
	err := db.Migrate(sl.cfg.DatabaseURL, sl.logger)
	if err != nil {
		return err
	}

	sl.logger.Info("initializing postgres storage")
	pgdb, err := db.NewPostgresDB(sl.cfg.DatabaseURL)
	if err != nil {
		return err
	}

	sl.stor = postgres.NewPgStorageManager(pgdb)
	return nil
}

func (sl *GophermartApp) initIdentityProvider() {
	sl.logger.Info("initializing identity provider")
	sl.idp = auth.NewIdentityProvider(&sl.cfg.AuthConfig, sl.stor.GetUserStorage(), sl.logger)
}

func (sl *GophermartApp) initRestServer() {
	sl.logger.Info("initializing REST server")

	userService := service.NewUserService(sl.stor.GetUserStorage(), sl.idp, sl.logger)
	orderService := service.NewLoyaltyService(sl.stor.GetLoyaltyStorage(), sl.logger)

	sl.restServer = handlers.NewRestServer(sl.cfg, sl.logger,
		handlers.NewUserManagementHandlers(userService),
		handlers.NewLoyaltyManagementHandlers(orderService, sl.idp, sl.logger),
	)
}

func (sl *GophermartApp) initAccrualIntegrationFlow() {
	sl.logger.Info("initializing accrual system integration")
	sl.accrualIntFlow = accrual.NewIntegrationFlow(&sl.cfg.AccrualIntegrationConfig,
		sl.stor.GetLoyaltyStorage(), sl.logger)
}
