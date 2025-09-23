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

type LoyaltyService struct {
	storage repository.LoyaltyStorage
	logger  *zap.SugaredLogger
}

func NewLoyaltyService(
	ls repository.LoyaltyStorage,
	l *zap.Logger,
) *LoyaltyService {
	return &LoyaltyService{
		storage: ls,
		logger:  logging.ComponentLogger(l, "order-management"),
	}
}

// RegisterOrder checks if order is already created, and if not - creates new order in system.
// Otherwise, returns error depending on user for which existing order is registered
func (s *LoyaltyService) RegisterOrder(ctx context.Context, userID uuid.UUID, orderID string) error {
	if !utils.IsValidLuhnNumber(orderID) {
		return ErrInvalidOrderID
	}

	existingOrder, err := s.storage.GetOrderByID(ctx, orderID)
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

	err = s.storage.CreateNewOrder(ctx, orderID, userID)
	if err != nil {
		return fmt.Errorf("error creating new order: %w", err)
	}
	return nil
}

func (s *LoyaltyService) GetOrdersList(ctx context.Context, userID uuid.UUID) ([]*model.Order, error) {
	orders, err := s.storage.GetOrdersByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("unable to read orders data: %w", err)
	}
	return orders, nil
}

func (s *LoyaltyService) RegisterWithdrawal(ctx context.Context, userID uuid.UUID, wdOrder *model.WithdrawOrder) error {
	if !utils.IsValidLuhnNumber(wdOrder.OrderID) {
		return ErrInvalidOrderID
	}

	err := s.storage.TryCreateWithdrawal(ctx, wdOrder.OrderID, userID, wdOrder.Sum)
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

func (s *LoyaltyService) GetWithdrawalsList(ctx context.Context, userID uuid.UUID) ([]*model.Withdrawal, error) {
	wds, err := s.storage.GetWithdrawalsByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("unable to read withdrawals data: %w", err)
	}
	return wds, nil
}

func (s *LoyaltyService) GetUserBalance(ctx context.Context, userID uuid.UUID) (*model.Balance, error) {
	balance, err := s.storage.GetUserBalance(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("unable to read balance data: %w", err)
	}
	return balance, nil
}
