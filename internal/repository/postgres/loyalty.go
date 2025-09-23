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

func (ls *LoyaltyPgStorage) UpdateOrderAccrual(
	ctx context.Context,
	orderAccrual *model.OrderAccrual,
	timestamp time.Time,
) error {
	tx, err := ls.pgdb.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer ls.pgdb.RollbackTx(ctx, tx)

	order, err := ls.txGetOrderByID(ctx, tx, orderAccrual.OrderID)
	if err != nil {
		return err
	}
	if err = ls.txSetOrderAccrual(ctx, tx, orderAccrual, timestamp); err != nil {
		return err
	}

	if orderAccrual.Accrual != 0.0 {
		if err = ls.txUpdateUserBalance(ctx, tx, order.UserID); err != nil {
			return err
		}
	}

	ls.pgdb.CommitTx(ctx, tx)
	return nil
}

func (ls *LoyaltyPgStorage) TryCreateWithdrawal(
	ctx context.Context,
	wdID string,
	userID uuid.UUID,
	amount float64,
) error {
	tx, err := ls.pgdb.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer ls.pgdb.RollbackTx(ctx, tx)

	// lock user balance - exclusive operation
	err = ls.txLockUserBalance(ctx, tx, userID)
	if err != nil {
		return err
	}

	// check if withdrawal with given ID already exists
	exists, err := ls.CheckWithdrawalExists(ctx, tx, wdID)
	if err != nil {
		return err
	}
	if exists {
		return common.ErrDuplicateEntity
	}

	// check aggregated balance for user
	balance, err := ls.txGetBalance(ctx, tx, userID)
	if err != nil {
		return err
	}
	if balance.Current < amount {
		return common.ErrInsufficientBalance
	}

	// create new withdrawal
	err = ls.txCreateWithdrawal(ctx, tx, wdID, userID, amount)
	if err != nil {
		return err
	}

	// update balance after new withdrawal
	err = ls.txUpdateUserBalance(ctx, tx, userID)
	if err != nil {
		return err
	}

	ls.pgdb.CommitTx(ctx, tx)
	return nil
}

func (ls *LoyaltyPgStorage) txUpdateUserBalance(
	ctx context.Context,
	tx pgx.Tx,
	userID uuid.UUID,
) error {
	accrued, err := ls.txFetchAccruedTotal(ctx, tx, userID)
	if err != nil {
		return err
	}
	withdrawn, err := ls.txFetchWithdrawnTotal(ctx, tx, userID)
	if err != nil {
		return err
	}
	return ls.txUpdateBalance(ctx, tx, userID, accrued, withdrawn)
}
