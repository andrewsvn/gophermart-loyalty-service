package repository

import (
	"context"
	"time"

	"github.com/andrewsvn/gophermart-ls/internal/db"
	"github.com/andrewsvn/gophermart-ls/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Facade struct {
	pgdb *db.PostgresDB

	userRepo       *userRepository
	orderRepo      *orderRepository
	withdrawalRepo *withdrawalRepository
	balanceRepo    *balanceRepository
}

func NewFacade(db *db.PostgresDB) *Facade {
	return &Facade{
		pgdb:           db,
		userRepo:       newUserRepository(db),
		orderRepo:      newOrderRepository(db),
		withdrawalRepo: newWithdrawalRepository(db),
		balanceRepo:    newBalanceRepository(db),
	}
}

// user management functions

func (r *Facade) GetUserByLogin(ctx context.Context, login string) (*model.User, error) {
	return r.userRepo.getUserByLogin(ctx, login)
}

func (r *Facade) GetUserByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	return r.userRepo.getUserByID(ctx, id)
}

func (r *Facade) CreateUser(ctx context.Context, login, authHash string) (*uuid.UUID, error) {
	return r.userRepo.createUser(ctx, login, authHash)
}

func (r *Facade) UpdateUserLoginTS(ctx context.Context, userID uuid.UUID) (bool, error) {
	return r.userRepo.updateUserLoginTS(ctx, userID)
}

func (r *Facade) CheckUserExistsByLogin(ctx context.Context, login string) (bool, error) {
	return r.userRepo.checkUserExistsByLogin(ctx, login)
}

// order management functions

func (r *Facade) GetOrderByID(ctx context.Context, orderID string) (*model.Order, error) {
	return r.orderRepo.getOrder(ctx, nil, orderID)
}

func (r *Facade) GetOrdersByUserID(ctx context.Context, userID uuid.UUID) ([]*model.Order, error) {
	return r.orderRepo.getOrdersByUserID(ctx, userID)
}

func (r *Facade) CreateNewOrder(ctx context.Context, orderID string, userID uuid.UUID) error {
	return r.orderRepo.createNewOrder(ctx, orderID, userID)
}

func (r *Facade) UpdateOrderAccrual(
	ctx context.Context,
	orderAccrual *model.OrderAccrual,
	timestamp time.Time,
) error {
	tx, err := r.pgdb.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer r.pgdb.RollbackTx(ctx, tx)

	order, err := r.orderRepo.getOrder(ctx, tx, orderAccrual.OrderID)
	if err != nil {
		return err
	}
	if err = r.orderRepo.setOrderAccrual(ctx, tx, orderAccrual, timestamp); err != nil {
		return err
	}

	if orderAccrual.Accrual != 0.0 {
		if err = r.updateUserBalance(ctx, tx, order.UserID); err != nil {
			return err
		}
	}

	r.pgdb.CommitTx(ctx, tx)
	return nil
}

func (r *Facade) FetchOrderIDsForUpdate(ctx context.Context, limit uint64) ([]string, error) {
	return r.orderRepo.fetchOrderIDsForUpdate(ctx, limit)
}

func (r *Facade) ResetPendingOrders(ctx context.Context) error {
	return r.orderRepo.resetPendingOrders(ctx)
}

// withdrawal management functions

func (r *Facade) GetWithdrawalByID(ctx context.Context, wdID string) (*model.Withdrawal, error) {
	return r.withdrawalRepo.GetWithdrawalByID(ctx, wdID)
}

func (r *Facade) GetWithdrawalsByUserID(ctx context.Context, userID uuid.UUID) ([]*model.Withdrawal, error) {
	return r.withdrawalRepo.GetWithdrawalsByUserID(ctx, userID)
}

func (r *Facade) TryCreateWithdrawal(
	ctx context.Context,
	wdID string,
	userID uuid.UUID,
	amount float64,
) error {
	tx, err := r.pgdb.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer r.pgdb.RollbackTx(ctx, tx)

	// lock user balance - exclusive operation
	err = r.balanceRepo.lockUserBalance(ctx, tx, userID)
	if err != nil {
		return err
	}

	// check if withdrawal with given ID already exists
	exists, err := r.withdrawalRepo.checkWithdrawalExists(ctx, tx, wdID)
	if err != nil {
		return err
	}
	if exists {
		return ErrDuplicateEntity
	}

	// check aggregated balance for user
	balance, err := r.balanceRepo.getBalanceByUser(ctx, tx, userID)
	if err != nil {
		return err
	}
	if balance.Current < amount {
		return ErrInsufficientBalance
	}

	// create new withdrawal
	err = r.withdrawalRepo.createWithdrawal(ctx, tx, wdID, userID, amount)
	if err != nil {
		return err
	}

	// update balance after new withdrawal
	err = r.updateUserBalance(ctx, tx, userID)
	if err != nil {
		return err
	}

	r.pgdb.CommitTx(ctx, tx)
	return nil
}

// balance management functions

func (r *Facade) GetUserBalance(ctx context.Context, userID uuid.UUID) (*model.Balance, error) {
	return r.balanceRepo.getBalanceByUser(ctx, nil, userID)
}

// updateUserBalance is in facade to reduce cohesion since it uses different repos
func (r *Facade) updateUserBalance(
	ctx context.Context,
	tx pgx.Tx,
	userID uuid.UUID,
) error {
	accrued, err := r.orderRepo.fetchAccruedSum(ctx, tx, userID)
	if err != nil {
		return err
	}
	withdrawn, err := r.withdrawalRepo.fetchWithdrawnSum(ctx, tx, userID)
	if err != nil {
		return err
	}
	return r.balanceRepo.updateBalance(ctx, tx, userID, accrued, withdrawn)
}
