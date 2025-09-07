package service

import (
	"context"

	"github.com/andrewsvn/gophermart-ls/internal/logging"
	"github.com/andrewsvn/gophermart-ls/internal/repository"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type OrderService struct {
	orderRepo *repository.OrderRepository
	userRepo  *repository.UserRepository
	logger    *zap.SugaredLogger
}

func NewOrderService(or *repository.OrderRepository, ur *repository.UserRepository, l *zap.Logger) *OrderService {
	return &OrderService{
		orderRepo: or,
		userRepo:  ur,
		logger:    logging.ComponentLogger(l, "order-management"),
	}
}

func (s *OrderService) RegisterOrder(ctx context.Context, orderId string, userId uuid.UUID) error {
	// TODO
	return nil
}
