package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/andrewsvn/gophermart-ls/internal/model"
	"github.com/andrewsvn/gophermart-ls/internal/repository/common"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (ls *LoyaltyPgStorage) GetUserBalance(ctx context.Context, userID uuid.UUID) (*model.Balance, error) {
	return ls.txGetBalance(ctx, nil, userID)
}

func (ls *LoyaltyPgStorage) txGetBalance(ctx context.Context, tx pgx.Tx, userID uuid.UUID) (*model.Balance, error) {
	sqlQuery, args, err := sqrl.
		Select("BALANCE, WITHDRAWN").
		From(balanceTableName).
		Where(squirrel.Eq{"USER_ID": userID}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidQuery, err)
	}

	rows, err := ls.query(ctx, tx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("%w %s: %v", ErrExecuteSelect, balanceTableName, err)
	}
	defer rows.Close()

	balance := &model.Balance{}
	if !rows.Next() {
		return balance, nil
	}

	if err = rows.Scan(&balance.Current, &balance.Withdrawn); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrScanningRow, err)
	}
	return balance, nil
}

func (ls *LoyaltyPgStorage) txUpdateBalance(
	ctx context.Context,
	tx pgx.Tx,
	userID uuid.UUID,
	accrued float64,
	withdrawn float64,
) error {
	sqlQuery, args, err := sqrl.
		Insert(balanceTableName).
		Columns("USER_ID", "BALANCE", "WITHDRAWN", "LAST_UPDATE_TS").
		Values(userID, accrued-withdrawn, withdrawn, time.Now()).
		Suffix(`ON CONFLICT (USER_ID) DO UPDATE
					SET BALANCE = EXCLUDED.BALANCE,
						WITHDRAWN = EXCLUDED.WITHDRAWN,
						LAST_UPDATE_TS = EXCLUDED.LAST_UPDATE_TS`).
		ToSql()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidQuery, err)
	}

	res, err := ls.exec(ctx, tx, sqlQuery, args...)
	if err != nil {
		return fmt.Errorf("%w %s: %v", ErrExecuteUpdate, balanceTableName, err)
	}
	if res.RowsAffected() == 0 {
		return fmt.Errorf("%w: balance is not updated for user %s", common.ErrEntityNotFound, userID)
	}
	return nil
}

func (ls *LoyaltyPgStorage) txLockUserBalance(ctx context.Context, tx pgx.Tx, userID uuid.UUID) error {
	sqlQuery, args, err := sqrl.
		Select("USER_ID").
		From(balanceTableName).
		Where(squirrel.Eq{"USER_ID": userID}).
		Suffix("FOR UPDATE").
		ToSql()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidQuery, err)
	}

	rows, err := ls.query(ctx, tx, sqlQuery, args...)
	if err != nil {
		return fmt.Errorf("unable to lock user balance: %v", err)
	}
	rows.Close()
	return nil
}
