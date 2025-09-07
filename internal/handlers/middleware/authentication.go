package middleware

import (
	"net/http"

	"github.com/andrewsvn/gophermart-ls/internal/auth"
	"github.com/andrewsvn/gophermart-ls/internal/logging"
	"go.uber.org/zap"
)

type Authentication struct {
	idp    *auth.IdentityProvider
	logger *zap.SugaredLogger
}

func NewAuthentication(idp *auth.IdentityProvider, l *zap.Logger) *Authentication {
	return &Authentication{
		idp:    idp,
		logger: logging.ComponentLogger(l, "authentication"),
	}
}

func (h *Authentication) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// TODO
		h.logger.Infow("authentication complete")
		next.ServeHTTP(w, r)
	})
}
