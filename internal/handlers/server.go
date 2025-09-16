package handlers

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/andrewsvn/gophermart-ls/internal/config"
	"github.com/andrewsvn/gophermart-ls/internal/handlers/middleware"
	"github.com/andrewsvn/gophermart-ls/internal/logging"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

type RouteProvider interface {
	RegisterRoutes(r chi.Router)
	SetHandlersLogger(hl *zap.SugaredLogger)
}

const (
	baseAPIPath = "/api/user"
)

type RestServer struct {
	httpSrv *http.Server
	logger  *zap.SugaredLogger
}

func NewRestServer(cfg *config.ServerConfig, l *zap.Logger, providers ...RouteProvider) *RestServer {
	logger := logging.ComponentLogger(l, "rest-server")

	r := chi.NewRouter()
	// common middleware
	r.Use(middleware.NewHTTPLogging(l).Middleware)

	r.Route(baseAPIPath, func(r chi.Router) {
		for _, provider := range providers {
			provider.RegisterRoutes(r)
			provider.SetHandlersLogger(logger)
		}
	})

	return &RestServer{
		httpSrv: &http.Server{
			Addr:    strings.Trim(cfg.URL, "\""),
			Handler: r,
		},
		logger: logger,
	}
}

func (rs *RestServer) Start() {
	go func() {
		rs.logger.Infow("starting http server",
			"address", rs.httpSrv.Addr,
		)
		if err := rs.httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			rs.logger.Fatal("failed to start http server", zap.Error(err))
		}
	}()
}

func (rs *RestServer) GracefulShutdown() {
	rs.logger.Info("shutting down http server...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second) // TODO: configurable
	defer cancel()
	if err := rs.httpSrv.Shutdown(ctx); err != nil {
		rs.logger.Error("failed to shutdown http server", zap.Error(err))
		return
	}
	rs.logger.Info("http server shut down successfully")
}
