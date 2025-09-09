package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/andrewsvn/gophermart-ls/internal/auth"
	"github.com/andrewsvn/gophermart-ls/internal/logging"
	"go.uber.org/zap"
)

const (
	authorizationHeaderName = "Authorization"
	authorizationType       = "Bearer"

	AuthorizedUserIDVar = "userID"
)

type Authorization struct {
	idp    *auth.IdentityProvider
	logger *zap.SugaredLogger
}

func NewAuthorization(idp *auth.IdentityProvider, l *zap.Logger) *Authorization {
	return &Authorization{
		idp:    idp,
		logger: logging.ComponentLogger(l, "authentication"),
	}
}

func (a *Authorization) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get(authorizationHeaderName)
		if authHeader == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != authorizationType {
			http.Error(w, "invalid authorization header format", http.StatusUnauthorized)
			return
		}

		userID, err := a.idp.AuthorizeUser(r.Context(), parts[1])
		if err != nil {
			if errors.Is(err, auth.ErrInvalidToken) {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}
			a.logger.Errorw("error while authorizing token", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}

		a.logger.Debugw("authorizing user", "user", *userID)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), AuthorizedUserIDVar, *userID)))
	})
}
