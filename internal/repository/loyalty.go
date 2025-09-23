package repository

import (
	"context"
	"time"

	"github.com/andrewsvn/gophermart-ls/internal/model"
	"github.com/google/uuid"
)

type LoyaltyStorage interface {
	GetOrderByID(ctx context.Context, orderID string) (*model.Order, error)
	GetOrdersByUserID(ctx context.Context, userID uuid.UUID) ([]*model.Order, error)
	CreateNewOrder(ctx context.Context, orderID string, userID uuid.UUID) error
	UpdateOrderAccrual(ctx context.Context, orderAccrual *model.OrderAccrual, timestamp time.Time) error
	FetchOrderIDsForUpdate(ctx context.Context, limit uint64) ([]string, error)
	ResetPendingOrders(ctx context.Context) error

	GetWithdrawalByID(ctx context.Context, wdID string) (*model.Withdrawal, error)
	GetWithdrawalsByUserID(ctx context.Context, userID uuid.UUID) ([]*model.Withdrawal, error)
	TryCreateWithdrawal(ctx context.Context, wdID string, userID uuid.UUID, amount float64) error

	GetUserBalance(ctx context.Context, userID uuid.UUID) (*model.Balance, error)
}
