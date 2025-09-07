package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/andrewsvn/gophermart-ls/internal/auth"
	"github.com/andrewsvn/gophermart-ls/internal/handlers/middleware"
	"github.com/andrewsvn/gophermart-ls/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type OrderManagementHandlers struct {
	orderService  *service.OrderService
	compressor    *middleware.Compressing
	authenticator *middleware.Authentication
	logger        *zap.SugaredLogger
}

type Balance struct {
	Current   float64 `json:"current"`
	Withdrawn float64 `json:"withdrawn"`
}

func NewOrderManagementHandlers(
	os *service.OrderService,
	idp *auth.IdentityProvider,
	baseLogger *zap.Logger,
) *OrderManagementHandlers {
	return &OrderManagementHandlers{
		orderService:  os,
		compressor:    middleware.NewCompressing(baseLogger),
		authenticator: middleware.NewAuthentication(idp, baseLogger),
	}
}

func (h *OrderManagementHandlers) RegisterRoutes(r chi.Router) {
	r.With(h.authenticator.Middleware).Route("/orders", func(r chi.Router) {
		r.Post("/", h.newOrderHandler())
		r.With(h.compressor.Middleware).Get("/", h.getOrdersHandler())
	})
	r.With(h.authenticator.Middleware).Route("/balance", func(r chi.Router) {
		r.Get("/", h.getBalanceHandler())
		r.Route("/withdraw", func(r chi.Router) {
			r.Post("/", h.withdrawHandler())
		})
	})
	r.With(h.authenticator.Middleware).Route("/withdrawals", func(r chi.Router) {
		r.With(h.compressor.Middleware).Get("/", h.getWithdrawalsHandler())
	})
}

func (h *OrderManagementHandlers) SetHandlersLogger(logger *zap.SugaredLogger) {
	h.logger = logger
}

func (h *OrderManagementHandlers) newOrderHandler() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		// TODO
		rw.WriteHeader(http.StatusCreated)
		_, err := rw.Write([]byte("Order created"))
		if err != nil {
			h.logger.Errorw("Failed to write response", "error", err)
		}
	}
}

func (h *OrderManagementHandlers) getOrdersHandler() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		// TODO
		rw.WriteHeader(http.StatusNoContent)
	}
}

func (h *OrderManagementHandlers) getBalanceHandler() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		// TODO
		balance := &Balance{}
		payload, err := json.Marshal(balance)
		if err != nil {
			h.logger.Errorw("Failed to marshal payload", "error", err)
			http.Error(rw, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		_, err = rw.Write(payload)
		if err != nil {
			h.logger.Errorw("Failed to write response", "error", err)
		}
	}
}

func (h *OrderManagementHandlers) withdrawHandler() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		// TODO
		rw.WriteHeader(http.StatusPaymentRequired)
	}
}

func (h *OrderManagementHandlers) getWithdrawalsHandler() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		// TODO
		rw.WriteHeader(http.StatusNoContent)
	}
}

func (h *OrderManagementHandlers) getUserId(r *http.Request) uuid.UUID {
	return r.Context().Value("userID").(uuid.UUID)
}
