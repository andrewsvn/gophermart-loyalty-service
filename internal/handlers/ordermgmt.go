package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/andrewsvn/gophermart-ls/internal/auth"
	"github.com/andrewsvn/gophermart-ls/internal/handlers/middleware"
	"github.com/andrewsvn/gophermart-ls/internal/handlers/utils"
	"github.com/andrewsvn/gophermart-ls/internal/model"
	"github.com/andrewsvn/gophermart-ls/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type LoyaltyManagementHandlers struct {
	loyaltyService *service.LoyaltyService
	compressor     *middleware.Compressing
	authenticator  *middleware.Authorization
	logger         *zap.SugaredLogger
}

func NewLoyaltyManagementHandlers(
	ls *service.LoyaltyService,
	idp *auth.IdentityProvider,
	baseLogger *zap.Logger,
) *LoyaltyManagementHandlers {
	return &LoyaltyManagementHandlers{
		loyaltyService: ls,
		compressor:     middleware.NewCompressing(baseLogger),
		authenticator:  middleware.NewAuthorization(idp, baseLogger),
	}
}

func (h *LoyaltyManagementHandlers) RegisterRoutes(r chi.Router) {
	authR := r.With(h.authenticator.Middleware)

	authR.Route("/orders", func(r chi.Router) {
		r.Post("/", utils.WithUserID(h.newOrderHandlerFunc))
		r.With(h.compressor.Middleware).Get("/", utils.WithUserID(h.getOrdersHandlerFunc))
	})
	authR.Route("/balance", func(r chi.Router) {
		r.Get("/", utils.WithUserID(h.getBalanceHandlerFunc))
		r.Route("/withdraw", func(r chi.Router) {
			r.Post("/", utils.WithUserID(h.withdrawHandlerFunc))
		})
	})
	authR.Route("/withdrawals", func(r chi.Router) {
		r.With(h.compressor.Middleware).Get("/", utils.WithUserID(h.getWithdrawalsHandlerFunc))
	})
}

func (h *LoyaltyManagementHandlers) SetHandlersLogger(logger *zap.SugaredLogger) {
	h.logger = logger
}

func (h *LoyaltyManagementHandlers) newOrderHandlerFunc(rw http.ResponseWriter, r *http.Request, userID uuid.UUID) {
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

	ctx := r.Context()
	h.logger.Debugw("registering new order", "userID", userID, "orderID", orderID)
	err = h.loyaltyService.RegisterOrder(ctx, userID, orderID)
	if err != nil {
		if errors.Is(err, service.ErrInvalidOrderID) {
			http.Error(rw, err.Error(), http.StatusUnprocessableEntity)
			return
		}
		if errors.Is(err, service.ErrOrderExistsForOtherUser) {
			http.Error(rw, err.Error(), http.StatusConflict)
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

	h.logger.Debugw("order registered", "userID", userID, "orderID", orderID)
	rw.WriteHeader(http.StatusAccepted)
	_, err = rw.Write([]byte("order accepted into processing"))
	if err != nil {
		h.logger.Errorw("failed to write response", "error", err)
	}
}

func (h *LoyaltyManagementHandlers) getOrdersHandlerFunc(rw http.ResponseWriter, r *http.Request, userID uuid.UUID) {
	ctx := r.Context()
	h.logger.Debugw("getting orders list", "userID", userID)
	orders, err := h.loyaltyService.GetOrdersList(ctx, userID)
	if err != nil {
		h.logger.Errorw("error getting orders list", "error", err)
		http.Error(rw, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}

	h.logger.Debugw("orders fetched",
		"userID", userID,
		"count", len(orders))
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
	h.writeJSONPayload(rw, http.StatusOK, payload)
}

func (h *LoyaltyManagementHandlers) getBalanceHandlerFunc(rw http.ResponseWriter, r *http.Request, userID uuid.UUID) {
	ctx := r.Context()
	h.logger.Debugw("getting balance", "userID", userID)
	balance, err := h.loyaltyService.GetUserBalance(ctx, userID)
	if err != nil {
		h.logger.Errorw("error getting user balance", "error", err)
		http.Error(rw, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	h.logger.Debugw("balance fetched")
	payload, err := json.Marshal(balance)
	if err != nil {
		h.logger.Errorw("failed to marshal payload", "error", err)
		http.Error(rw, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	h.writeJSONPayload(rw, http.StatusOK, payload)
}

func (h *LoyaltyManagementHandlers) withdrawHandlerFunc(rw http.ResponseWriter, r *http.Request, userID uuid.UUID) {
	wdOrder := &model.WithdrawOrder{}
	err := json.NewDecoder(r.Body).Decode(wdOrder)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
	}

	if wdOrder.OrderID == "" {
		http.Error(rw, "orderID is required", http.StatusBadRequest)
		return
	}
	if wdOrder.Sum <= 0 {
		http.Error(rw, "sum must be a positive numeric value", http.StatusBadRequest)
	}

	ctx := r.Context()
	h.logger.Debugw("creating new withdrawal", "userID", userID, "orderID", wdOrder.OrderID)
	err = h.loyaltyService.RegisterWithdrawal(ctx, userID, wdOrder)
	if err != nil {
		if errors.Is(err, service.ErrInvalidOrderID) {
			http.Error(rw, err.Error(), http.StatusUnprocessableEntity)
			return
		}
		if errors.Is(err, service.ErrNotEnoughBalance) {
			http.Error(rw, err.Error(), http.StatusPaymentRequired)
			return
		}
		if errors.Is(err, service.ErrWithdrawalAlreadyExists) {
			http.Error(rw, err.Error(), http.StatusConflict)
			return
		}
		h.logger.Errorw("error registering withdrawal", "error", err)
		http.Error(rw, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	h.logger.Debugw("withdrawal created", "userID", userID, "orderID", wdOrder.OrderID)
	rw.WriteHeader(http.StatusOK)
}

func (h *LoyaltyManagementHandlers) getWithdrawalsHandlerFunc(rw http.ResponseWriter, r *http.Request, userID uuid.UUID) {
	ctx := r.Context()
	h.logger.Debugw("getting withdrawals list", "userID", userID)
	withdraws, err := h.loyaltyService.GetWithdrawalsList(ctx, userID)
	if err != nil {
		h.logger.Errorw("error getting withdrawals list", "error", err)
		http.Error(rw, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	h.logger.Debugw("withdrawals fetched",
		"userID", userID,
		"count", len(withdraws))
	payload, err := json.Marshal(withdraws)
	if err != nil {
		h.logger.Errorw("failed to marshal payload", "error", err)
		http.Error(rw, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	h.writeJSONPayload(rw, http.StatusOK, payload)
}

func (h *LoyaltyManagementHandlers) writeJSONPayload(rw http.ResponseWriter, httpCode int, payload []byte) {
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(httpCode)
	_, err := rw.Write(payload)
	if err != nil {
		h.logger.Errorw("failed to write response", "error", err)
	}
}
