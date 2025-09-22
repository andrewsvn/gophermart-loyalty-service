package postgres

import (
	"context"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/andrewsvn/gophermart-ls/internal/model"
	"github.com/andrewsvn/gophermart-ls/internal/repository/common"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (r *LoyaltyPgStorage) GetWithdrawalByID(ctx context.Context, wdID string) (*model.Withdrawal, error) {
	sqlQuery, args, err := sqrl.
		Select(withdrawalColumns).
		From(withdrawalTableName).
		Where(squirrel.Eq{"ID": wdID}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidQuery, err)
	}

	rows, err := r.query(ctx, nil, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("%w %s: %v", ErrExecuteSelect, withdrawalTableName, err)
	}
	defer rows.Close()

	return r.withdrawalFromRow(rows)
}

func (r *LoyaltyPgStorage) GetWithdrawalsByUserID(
	ctx context.Context,
	userID uuid.UUID,
) ([]*model.Withdrawal, error) {
	sqlQuery, args, err := sqrl.
		Select(withdrawalColumns).
		From(withdrawalTableName).
		Where(squirrel.Eq{"USER_ID": userID}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidQuery, err)
	}

	rows, err := r.query(ctx, nil, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("%w %s: %v", ErrExecuteSelect, withdrawalTableName, err)
	}
	defer rows.Close()

	return r.withdrawalsFromRows(rows)
}

func (r *LoyaltyPgStorage) txFetchWithdrawnTotal(ctx context.Context, tx pgx.Tx, userID uuid.UUID) (float64, error) {
	sqlQuery, args, err := sqrl.
		Select("COALESCE(SUM(AMOUNT), 0)").
		From(withdrawalTableName).
		Where(squirrel.Eq{"USER_ID": userID}).
		ToSql()
	if err != nil {
		return 0, err
	}

	rows, err := r.query(ctx, tx, sqlQuery, args...)
	if err != nil {
		return 0, fmt.Errorf("%w %s: %v", ErrExecuteSelect, withdrawalTableName, err)
	}
	defer rows.Close()

	if !rows.Next() {
		return 0, nil
	}

	var total float64
	if err := rows.Scan(&total); err != nil {
		return 0, fmt.Errorf("%w %s: %v", ErrScanningRow, withdrawalTableName, err)
	}
	return total, nil
}

func (r *LoyaltyPgStorage) CheckWithdrawalExists(ctx context.Context, tx pgx.Tx, wdID string) (bool, error) {
	sqlQuery, args, err := sqrl.
		Select("ID").
		From(withdrawalTableName).
		Where(squirrel.Eq{"ID": wdID}).
		ToSql()
	if err != nil {
		return false, fmt.Errorf("%w: %v", ErrInvalidQuery, err)
	}

	rows, err := r.query(ctx, tx, sqlQuery, args...)
	if err != nil {
		return false, fmt.Errorf("%w %s: %v", ErrExecuteSelect, withdrawalTableName, err)
	}
	defer rows.Close()
	return rows.Next(), nil
}

func (r *LoyaltyPgStorage) txCreateWithdrawal(
	ctx context.Context,
	tx pgx.Tx,
	wdID string,
	userID uuid.UUID,
	amount float64,
) error {
	sqlQuery, args, err := sqrl.
		Insert(withdrawalTableName).
		Columns("ID, USER_ID, AMOUNT").
		Values(wdID, userID, amount).
		ToSql()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidQuery, err)
	}

	_, err = r.exec(ctx, tx, sqlQuery, args...)
	if err != nil {
		return fmt.Errorf("%w %s: %w", ErrExecuteInsert, withdrawalTableName, err)
	}
	return nil
}

func (r *LoyaltyPgStorage) withdrawalFromRow(rows pgx.Rows) (*model.Withdrawal, error) {
	if !rows.Next() {
		return nil, common.ErrEntityNotFound
	}
	return r.scanWithdrawal(rows)
}

func (r *LoyaltyPgStorage) withdrawalsFromRows(rows pgx.Rows) ([]*model.Withdrawal, error) {
	wds := make([]*model.Withdrawal, 0)
	for rows.Next() {
		wd, err := r.scanWithdrawal(rows)
		if err != nil {
			return nil, err
		}
		wds = append(wds, wd)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrFetchingRows, err)
	}

	return wds, nil
}

func (r *LoyaltyPgStorage) scanWithdrawal(rows pgx.Rows) (*model.Withdrawal, error) {
	wd := model.Withdrawal{}
	err := rows.Scan(&wd.ID, &wd.UserID, &wd.Sum, &wd.ProcessedAt)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrScanningRow, err)
	}
	return &wd, nil
}
