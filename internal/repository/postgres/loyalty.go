package postgres

import (
	"context"
	"time"

	"github.com/andrewsvn/gophermart-ls/internal/db"
	"github.com/andrewsvn/gophermart-ls/internal/model"
	"github.com/andrewsvn/gophermart-ls/internal/repository/common"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type LoyaltyPgStorage struct {
	basePgStorage
}

func NewLoyaltyPgStorage(pgdb *db.PostgresDB) *LoyaltyPgStorage {
	return &LoyaltyPgStorage{
		basePgStorage: basePgStorage{
			pgdb: pgdb,
		},
	}
}

func (r *LoyaltyPgStorage) UpdateOrderAccrual(
	ctx context.Context,
	orderAccrual *model.OrderAccrual,
	timestamp time.Time,
) error {
	tx, err := r.pgdb.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer r.pgdb.RollbackTx(ctx, tx)

	order, err := r.txGetOrderByID(ctx, tx, orderAccrual.OrderID)
	if err != nil {
		return err
	}
	if err = r.txSetOrderAccrual(ctx, tx, orderAccrual, timestamp); err != nil {
		return err
	}

	if orderAccrual.Accrual != 0.0 {
		if err = r.txUpdateUserBalance(ctx, tx, order.UserID); err != nil {
			return err
		}
	}

	r.pgdb.CommitTx(ctx, tx)
	return nil
}

func (r *LoyaltyPgStorage) TryCreateWithdrawal(
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
	err = r.txLockUserBalance(ctx, tx, userID)
	if err != nil {
		return err
	}

	// check if withdrawal with given ID already exists
	exists, err := r.CheckWithdrawalExists(ctx, tx, wdID)
	if err != nil {
		return err
	}
	if exists {
		return common.ErrDuplicateEntity
	}

	// check aggregated balance for user
	balance, err := r.txGetBalance(ctx, tx, userID)
	if err != nil {
		return err
	}
	if balance.Current < amount {
		return common.ErrInsufficientBalance
	}

	// create new withdrawal
	err = r.txCreateWithdrawal(ctx, tx, wdID, userID, amount)
	if err != nil {
		return err
	}

	// update balance after new withdrawal
	err = r.txUpdateUserBalance(ctx, tx, userID)
	if err != nil {
		return err
	}

	r.pgdb.CommitTx(ctx, tx)
	return nil
}

func (r *LoyaltyPgStorage) txUpdateUserBalance(
	ctx context.Context,
	tx pgx.Tx,
	userID uuid.UUID,
) error {
	accrued, err := r.txFetchAccruedTotal(ctx, tx, userID)
	if err != nil {
		return err
	}
	withdrawn, err := r.txFetchWithdrawnTotal(ctx, tx, userID)
	if err != nil {
		return err
	}
	return r.txUpdateBalance(ctx, tx, userID, accrued, withdrawn)
}
