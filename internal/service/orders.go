package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/andrewsvn/gophermart-ls/internal/logging"
	"github.com/andrewsvn/gophermart-ls/internal/model"
	"github.com/andrewsvn/gophermart-ls/internal/repository"
	"github.com/andrewsvn/gophermart-ls/internal/utils"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type OrderService struct {
	repoFacade *repository.Facade
	logger     *zap.SugaredLogger
}

var (
	ErrInvalidOrderID          = errors.New("order ID has incorrect format")
	ErrWithdrawalAlreadyExists = errors.New("withdrawal already exists")
	ErrOrderExistsForSameUser  = errors.New("order already exists for the same user")
	ErrOrderExistsForOtherUser = errors.New("order already exists for another user")
	ErrNotEnoughBalance        = errors.New("not enough loyalty points available")
)

func NewOrderService(
	rf *repository.Facade,
	l *zap.Logger,
) *OrderService {
	return &OrderService{
		repoFacade: rf,
		logger:     logging.ComponentLogger(l, "order-management"),
	}
}

// RegisterOrder checks if order is already created, and if not - creates new order in system.
// Otherwise, returns error depending on user for which existing order is registered
func (s *OrderService) RegisterOrder(ctx context.Context, userID uuid.UUID, orderID string) error {
	if !utils.IsValidLuhnNumber(orderID) {
		return ErrInvalidOrderID
	}

	existingOrder, err := s.repoFacade.GetOrderByID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("error getting existing order: %w", err)
	}
	if existingOrder != nil {
		if existingOrder.UserID == userID {
			return ErrOrderExistsForSameUser
		} else {
			return ErrOrderExistsForOtherUser
		}
	}

	err = s.repoFacade.CreateNewOrder(ctx, orderID, userID)
	if err != nil {
		return fmt.Errorf("error creating new order: %w", err)
	}
	return nil
}

func (s *OrderService) GetOrdersList(ctx context.Context, userID uuid.UUID) ([]*model.Order, error) {
	orders, err := s.repoFacade.GetOrdersByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("unable to read orders data: %w", err)
	}
	return orders, nil
}

func (s *OrderService) RegisterWithdrawal(ctx context.Context, userID uuid.UUID, wdOrder *model.WithdrawOrder) error {
	if !utils.IsValidLuhnNumber(wdOrder.OrderID) {
		return ErrInvalidOrderID
	}

	err := s.repoFacade.TryCreateWithdrawal(ctx, wdOrder.OrderID, userID, wdOrder.Sum)
	if err != nil {
		if errors.Is(err, repository.ErrInsufficientBalance) {
			return ErrNotEnoughBalance
		}
		if errors.Is(err, repository.ErrDuplicateEntity) {
			return ErrWithdrawalAlreadyExists
		}
		return fmt.Errorf("error creating withdrawal: %w", err)
	}
	return nil
}

func (s *OrderService) GetWithdrawalsList(ctx context.Context, userID uuid.UUID) ([]*model.Withdrawal, error) {
	wds, err := s.repoFacade.GetWithdrawalsByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("unable to read withdrawals data: %w", err)
	}
	return wds, nil
}

func (s *OrderService) GetUserBalance(ctx context.Context, userID uuid.UUID) (*model.Balance, error) {
	balance, err := s.repoFacade.GetUserBalance(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("unable to read balance data: %w", err)
	}
	return balance, nil
}
