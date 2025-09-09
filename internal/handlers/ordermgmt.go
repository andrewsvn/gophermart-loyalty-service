package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/andrewsvn/gophermart-ls/internal/auth"
	"github.com/andrewsvn/gophermart-ls/internal/handlers/middleware"
	"github.com/andrewsvn/gophermart-ls/internal/model"
	"github.com/andrewsvn/gophermart-ls/internal/service"
	"github.com/andrewsvn/gophermart-ls/internal/utils"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type OrderManagementHandlers struct {
	orderService  *service.OrderService
	compressor    *middleware.Compressing
	authenticator *middleware.Authorization
	logger        *zap.SugaredLogger
}

func NewOrderManagementHandlers(
	os *service.OrderService,
	idp *auth.IdentityProvider,
	baseLogger *zap.Logger,
) *OrderManagementHandlers {
	return &OrderManagementHandlers{
		orderService:  os,
		compressor:    middleware.NewCompressing(baseLogger),
		authenticator: middleware.NewAuthorization(idp, baseLogger),
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
		ctx := r.Context()
		userId, ok := h.getUserId(ctx)
		if !ok {
			http.Error(rw, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		orderIDBytes, err := io.ReadAll(r.Body)
		if err != nil {
			h.logger.Errorw("error reading request body", "error", err)
			http.Error(rw, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		orderID := strings.TrimSpace(string(orderIDBytes))
		if orderID == "" {
			http.Error(rw, "orderId is required", http.StatusBadRequest)
			return
		}
		if !utils.IsValidLuhnNumber(orderID) {
			http.Error(rw, "order ID has incorrect format", http.StatusUnprocessableEntity)
			return
		}

		h.logger.Debugw("registering new order", "userId", userId, "orderId", orderID)
		err = h.orderService.RegisterOrder(ctx, userId, orderID)
		if err != nil {
			if errors.Is(err, service.ErrOrderExistsForOtherUser) {
				http.Error(rw, "order already registered for other user", http.StatusConflict)
				return
			}
			if errors.Is(err, service.ErrOrderExistsForSameUser) {
				rw.WriteHeader(http.StatusOK)
				return
			}
			h.logger.Errorw("error registering new order", "error", err)
			http.Error(rw, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		h.logger.Debugw("order registered", "userId", userId, "orderId", orderID)
		rw.WriteHeader(http.StatusCreated)
		_, err = rw.Write([]byte("Order created"))
		if err != nil {
			h.logger.Errorw("failed to write response", "error", err)
		}
	}
}

func (h *OrderManagementHandlers) getOrdersHandler() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		userId, ok := h.getUserId(ctx)
		if !ok {
			http.Error(rw, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		orders, err := h.orderService.GetOrdersList(ctx, userId)
		if err != nil {
			h.logger.Errorw("error getting orders list", "error", err)
			http.Error(rw, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}

		if len(orders) == 0 {
			rw.WriteHeader(http.StatusNoContent)
			return
		}

		payload, err := json.Marshal(orders)
		if err != nil {
			h.logger.Errorw("failed to marshal payload", "error", err)
			http.Error(rw, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		_, err = rw.Write(payload)
		if err != nil {
			h.logger.Errorw("failed to write response", "error", err)
		}
	}
}

func (h *OrderManagementHandlers) getBalanceHandler() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		userId, ok := h.getUserId(ctx)
		if !ok {
			http.Error(rw, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		balance, err := h.orderService.GetUserBalance(ctx, userId)
		if err != nil {
			h.logger.Errorw("error getting user balance", "error", err)
			http.Error(rw, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		payload, err := json.Marshal(balance)
		if err != nil {
			h.logger.Errorw("failed to marshal payload", "error", err)
			http.Error(rw, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		_, err = rw.Write(payload)
		if err != nil {
			h.logger.Errorw("failed to write response", "error", err)
		}
	}
}

func (h *OrderManagementHandlers) withdrawHandler() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		userId, ok := h.getUserId(ctx)
		if !ok {
			http.Error(rw, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		wdOrder := &model.WithdrawOrder{}
		err := json.NewDecoder(r.Body).Decode(wdOrder)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusBadRequest)
		}

		if wdOrder.OrderID == "" {
			http.Error(rw, "orderID is required", http.StatusBadRequest)
			return
		}
		if !utils.IsValidLuhnNumber(wdOrder.OrderID) {
			http.Error(rw, "orderID has incorrect format", http.StatusUnprocessableEntity)
			return
		}

		if wdOrder.Sum <= 0 {
			http.Error(rw, "positive sum is required", http.StatusBadRequest)
		}

		balance, err := h.orderService.GetUserBalance(ctx, userId)
		if err != nil {
			h.logger.Errorw("error getting user balance", "error", err)
			http.Error(rw, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		if wdOrder.Sum > balance.Current {
			http.Error(rw, "not enough bonuses available", http.StatusPaymentRequired)
		}

		err = h.orderService.RegisterWithdrawal(ctx, userId, wdOrder)
		if err != nil {
			h.logger.Errorw("error registering withdrawal", "error", err)
			http.Error(rw, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		rw.WriteHeader(http.StatusOK)
	}
}

func (h *OrderManagementHandlers) getWithdrawalsHandler() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		userId, ok := h.getUserId(ctx)
		if !ok {
			http.Error(rw, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		withdraws, err := h.orderService.GetWithdrawalsList(ctx, userId)
		if err != nil {
			h.logger.Errorw("error getting withdrawals list", "error", err)
			http.Error(rw, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		payload, err := json.Marshal(withdraws)
		if err != nil {
			h.logger.Errorw("failed to marshal payload", "error", err)
			http.Error(rw, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		_, err = rw.Write(payload)
		if err != nil {
			h.logger.Errorw("failed to write response", "error", err)
		}
	}
}

func (h *OrderManagementHandlers) getUserId(ctx context.Context) (uuid.UUID, bool) {
	userId, ok := ctx.Value(middleware.AuthorizedUserIDVar).(uuid.UUID)
	if !ok {
		h.logger.Errorw("failed to get authenticated user id")
		return uuid.Nil, false
	}
	return userId, true
}
