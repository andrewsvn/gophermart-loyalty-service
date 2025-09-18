package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/andrewsvn/gophermart-ls/internal/db"
	"github.com/andrewsvn/gophermart-ls/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type balanceRepository struct {
	baseRepository
	pgdb *db.PostgresDB
}

func newBalanceRepository(db *db.PostgresDB) *balanceRepository {
	return &balanceRepository{
		baseRepository: baseRepository{
			pgdb: db,
		},
		pgdb: db,
	}
}

func (r *balanceRepository) getBalanceByUser(ctx context.Context, tx pgx.Tx, userID uuid.UUID) (*model.Balance, error) {
	sqlQuery, args, err := r.pgdb.Sqrl().
		Select("BALANCE, WITHDRAWN").
		From(balanceTableName).
		Where(squirrel.Eq{"USER_ID": userID}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidQuery, err)
	}

	rows, err := r.query(ctx, tx, sqlQuery, args...)
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

func (r *balanceRepository) updateBalance(
	ctx context.Context,
	tx pgx.Tx,
	userID uuid.UUID,
	accrued float64,
	withdrawn float64,
) error {
	sqlQuery, args, err := r.pgdb.Sqrl().
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

	res, err := r.exec(ctx, tx, sqlQuery, args...)
	if err != nil {
		return fmt.Errorf("%w %s: %v", ErrExecuteUpdate, balanceTableName, err)
	}
	if res.RowsAffected() == 0 {
		return fmt.Errorf("%w: balance is not updated for user %s", ErrEntityNotFound, userID)
	}
	return nil
}

func (r *balanceRepository) lockUserBalance(ctx context.Context, tx pgx.Tx, userID uuid.UUID) error {
	sqlQuery, args, err := r.pgdb.Sqrl().
		Select("USER_ID").
		From(balanceTableName).
		Where(squirrel.Eq{"USER_ID": userID}).
		Suffix("FOR UPDATE").
		ToSql()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidQuery, err)
	}

	rows, err := tx.Query(ctx, sqlQuery, args...)
	if err != nil {
		return fmt.Errorf("unable to lock user balance: %v", err)
	}
	rows.Close()
	return nil
}
