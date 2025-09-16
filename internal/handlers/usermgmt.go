package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/andrewsvn/gophermart-ls/internal/service"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

type UserManagementHandlers struct {
	userService *service.UserService
	logger      *zap.SugaredLogger
}

type UserLoginData struct {
	Login    string `json:"login,required"`
	Password string `json:"password,required"`
}

func NewUserManagementHandlers(us *service.UserService) *UserManagementHandlers {
	return &UserManagementHandlers{
		userService: us,
	}
}

func (h *UserManagementHandlers) RegisterRoutes(r chi.Router) {
	r.Route("/register", func(r chi.Router) {
		r.Post("/", h.registerUserHandler())
	})
	r.Route("/login", func(r chi.Router) {
		r.Post("/", h.loginUserHandler())
	})
}

func (h *UserManagementHandlers) SetHandlersLogger(logger *zap.SugaredLogger) {
	h.logger = logger
}

func (h *UserManagementHandlers) registerUserHandler() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		loginData := &UserLoginData{}
		err := json.NewDecoder(r.Body).Decode(loginData)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusBadRequest)
			return
		}

		if loginData.Login == "" || loginData.Password == "" {
			http.Error(rw, "login and/or password can't be empty", http.StatusBadRequest)
			return
		}

		err = h.userService.RegisterUser(r.Context(), loginData.Login, loginData.Password)
		if err != nil {
			if errors.Is(err, service.ErrUserAlreadyExists) {
				http.Error(rw, "user already exists", http.StatusConflict)
				return
			}
			h.logger.Errorw("failed to register user", "error", err)
			http.Error(rw, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		h.authorize(r.Context(), rw, loginData)
	}
}

func (h *UserManagementHandlers) loginUserHandler() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		loginData := &UserLoginData{}
		err := json.NewDecoder(r.Body).Decode(loginData)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusBadRequest)
			return
		}

		if loginData.Login == "" || loginData.Password == "" {
			http.Error(rw, "login and/or password can't be empty", http.StatusBadRequest)
			return
		}

		h.authorize(r.Context(), rw, loginData)
	}
}

func (h *UserManagementHandlers) authorize(
	ctx context.Context,
	rw http.ResponseWriter,
	loginData *UserLoginData,
) {
	authResult, err := h.userService.LoginUser(ctx, loginData.Login, loginData.Password)
	if err != nil {
		if errors.Is(err, service.ErrWrongLoginPassword) {
			http.Error(rw, err.Error(), http.StatusUnauthorized)
			return
		}
		h.logger.Errorw("failed to authorize user", "error", err)
		http.Error(rw, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	authCookie := &http.Cookie{
		Name:     "access_token",
		Value:    authResult.AccessToken,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	}
	http.SetCookie(rw, authCookie)

	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(http.StatusOK)
	err = json.NewEncoder(rw).Encode(authResult)
	if err != nil {
		h.logger.Errorw("failed to encode auth data", "error", err)
	}
}
