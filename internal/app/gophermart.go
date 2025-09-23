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

func (app *GophermartApp) Start() {
	app.logger.Info("starting gophermart-loyalty-service")
	app.restServer.Start()
	app.accrualIntFlow.Start()
}

func (app *GophermartApp) Stop() {
	wg := &sync.WaitGroup{}
	wg.Add(2)
	go func() {
		defer wg.Done()
		defer logging.Sync(app.logger)
		app.restServer.GracefulShutdown()
	}()
	go func() {
		defer wg.Done()
		defer logging.Sync(app.logger)
		app.accrualIntFlow.Shutdown()
	}()
	wg.Wait()

	app.stor.Close()
	app.logger.Info("gophermart-loyalty-service stopped")
}

func (app *GophermartApp) initStorage() error {
	app.logger.Info("migrating database schema")
	err := db.Migrate(app.cfg.DatabaseURL, app.logger)
	if err != nil {
		return err
	}

	app.logger.Info("initializing postgres storage")
	pgdb, err := db.NewPostgresDB(app.cfg.DatabaseURL)
	if err != nil {
		return err
	}

	app.stor = postgres.NewPgStorageManager(pgdb)
	return nil
}

func (app *GophermartApp) initIdentityProvider() {
	app.logger.Info("initializing identity provider")
	app.idp = auth.NewIdentityProvider(&app.cfg.AuthConfig, app.stor.GetUserStorage(), app.logger)
}

func (app *GophermartApp) initRestServer() {
	app.logger.Info("initializing REST server")

	userService := service.NewUserService(app.stor.GetUserStorage(), app.idp, app.logger)
	orderService := service.NewLoyaltyService(app.stor.GetLoyaltyStorage(), app.logger)

	app.restServer = handlers.NewRestServer(app.cfg, app.logger,
		handlers.NewUserManagementHandlers(userService),
		handlers.NewLoyaltyManagementHandlers(orderService, app.idp, app.logger),
	)
}

func (app *GophermartApp) initAccrualIntegrationFlow() {
	app.logger.Info("initializing accrual system integration")
	app.accrualIntFlow = accrual.NewIntegrationFlow(&app.cfg.AccrualIntegrationConfig,
		app.stor.GetLoyaltyStorage(), app.logger)
}
