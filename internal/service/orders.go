package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/andrewsvn/gophermart-ls/internal/logging"
	"github.com/andrewsvn/gophermart-ls/internal/model"
	"github.com/andrewsvn/gophermart-ls/internal/repository"
	"github.com/andrewsvn/gophermart-ls/internal/utils"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type OrderService struct {
	orderRepo      *repository.OrderRepository
	withdrawalRepo *repository.WithdrawalRepository
	userRepo       *repository.UserRepository
	logger         *zap.SugaredLogger
}

var (
	ErrInvalidOrderID          = errors.New("order ID has incorrect format")
	ErrOrderExistsForSameUser  = errors.New("order already exists for the same user")
	ErrOrderExistsForOtherUser = errors.New("order already exists for another user")
	ErrNotEnoughBalance        = errors.New("not enough loyalty points available")
)

func NewOrderService(
	or *repository.OrderRepository,
	wr *repository.WithdrawalRepository,
	l *zap.Logger,
) *OrderService {
	return &OrderService{
		orderRepo:      or,
		withdrawalRepo: wr,
		logger:         logging.ComponentLogger(l, "order-management"),
	}
}

// RegisterOrder checks if order is already created, and if not - creates new order in system.
// Otherwise, returns error depending on user for which existing order is registered
func (s *OrderService) RegisterOrder(ctx context.Context, userID uuid.UUID, orderID string) error {
	if !utils.IsValidLuhnNumber(orderID) {
		return ErrInvalidOrderID
	}

	existingOrder, err := s.orderRepo.GetOrderByID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("error getting existing order: %w", err)
	}
	if existingOrder != nil {
		if strings.ToLower(existingOrder.UserID) == strings.ToLower(userID.String()) {
			return ErrOrderExistsForSameUser
		} else {
			return ErrOrderExistsForOtherUser
		}
	}

	return s.orderRepo.CreateNewOrder(ctx, orderID, userID)
}

func (s *OrderService) GetOrdersList(ctx context.Context, userID uuid.UUID) ([]*model.Order, error) {
	orders, err := s.orderRepo.GetOrdersByUserId(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("unable to read orders data: %w", err)
	}
	return orders, nil
}

func (s *OrderService) RegisterWithdrawal(ctx context.Context, userID uuid.UUID, wdOrder *model.WithdrawOrder) error {
	if !utils.IsValidLuhnNumber(wdOrder.OrderID) {
		return ErrInvalidOrderID
	}

	err := s.withdrawalRepo.TryCreateWithdrawal(ctx, wdOrder.OrderID, userID, wdOrder.Sum)
	if err != nil {
		if errors.Is(err, repository.ErrInsufficientBalance) {
			return ErrNotEnoughBalance
		}
		if errors.Is(err, repository.ErrDuplicateEntity) {
			return ErrOrderExistsForSameUser
		}
		return fmt.Errorf("error creating withdrawal: %w", err)
	}
	return nil
}

func (s *OrderService) GetWithdrawalsList(ctx context.Context, userID uuid.UUID) ([]*model.Withdrawal, error) {
	wds, err := s.withdrawalRepo.GetWithdrawalsByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("unable to read withdrawals data: %w", err)
	}
	return wds, nil
}

func (s *OrderService) GetUserBalance(ctx context.Context, userID uuid.UUID) (*model.Balance, error) {
	accrued, err := s.orderRepo.GetTotalAccrualByUserId(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("unable to read orders data: %w", err)
	}
	withdrawn, err := s.withdrawalRepo.GetTotalWithdrawnByUserId(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("unable to read withdrawals data: %w", err)
	}

	return &model.Balance{
		Current:   accrued - withdrawn,
		Withdrawn: withdrawn,
	}, nil
}
