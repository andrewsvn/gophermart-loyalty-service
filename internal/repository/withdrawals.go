package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/andrewsvn/gophermart-ls/internal/db"
	"github.com/andrewsvn/gophermart-ls/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type WithdrawalRepository struct {
	baseRepository
}

const (
	withdrawalTableName = "LS_WITHDRAWALS"
	withdrawalColumns   = "ID, USER_ID, AMOUNT, CREATE_TS"
)

var (
	ErrWithdrawalIdNotUnique = errors.New("WithdrawalId is not unique")
	ErrInsufficientBalance   = errors.New("insufficient balance")
)

func NewWithdrawalRepository(db *db.PostgresDB) *WithdrawalRepository {
	return &WithdrawalRepository{
		baseRepository{
			db:        db,
			sqrl:      squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
			tableName: withdrawalTableName,
			columns:   withdrawalColumns,
		},
	}
}

func (r *WithdrawalRepository) GetWithdrawalByID(ctx context.Context, wdId string) (*model.Withdrawal, error) {
	rows, err := r.queryRows(ctx, func(sb squirrel.SelectBuilder) squirrel.SelectBuilder {
		return sb.Where(squirrel.Eq{"ID": wdId})
	})
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.fromRow(rows)
}

func (r *WithdrawalRepository) GetWithdrawalsByUserID(
	ctx context.Context,
	userID uuid.UUID,
) ([]*model.Withdrawal, error) {
	rows, err := r.queryRows(ctx, func(sb squirrel.SelectBuilder) squirrel.SelectBuilder {
		return sb.Where(squirrel.Eq{"USER_ID": userID})
	})
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.fromRows(rows)
}

func (r *WithdrawalRepository) GetTotalWithdrawnByUserId(
	ctx context.Context,
	userID uuid.UUID,
) (float64, error) {
	sqlQuery, args, err := r.sqrl.
		Select("COALESCE(SUM(AMOUNT), 0)").
		From(r.tableName).
		Where(squirrel.Eq{"USER_ID": userID}).
		ToSql()
	if err != nil {
		return 0, err
	}

	rows, err := r.db.Pool().Query(ctx, sqlQuery, args...)
	if err != nil {
		return 0, fmt.Errorf("error querying rows from table %s: %w", r.tableName, err)
	}
	defer rows.Close()

	if !rows.Next() {
		return 0, nil
	}

	var total float64
	if err := rows.Scan(&total); err != nil {
		return 0, fmt.Errorf("error scanning row from table %s: %w", r.tableName, err)
	}
	return total, nil
}

func (r *WithdrawalRepository) TryCreateWithdrawal(
	ctx context.Context,
	wdId string,
	userId uuid.UUID,
	amount float64,
) error {
	// check if withdrawal with given ID already exists
	sqlQuery, args, err := r.sqrl.
		Select("ID").
		From(r.tableName).
		Where(squirrel.Eq{"ID": wdId}).
		ToSql()
	if err != nil {
		return fmt.Errorf("error building query: %w", err)
	}

	rows, err := r.db.Pool().Query(ctx, sqlQuery, args...)
	if err != nil {
		return fmt.Errorf("error querying rows from table %s: %w", r.tableName, err)
	}

	if rows.Next() {
		return ErrWithdrawalIdNotUnique
	}

	// create withdrawal if balance is enough
	sqlQuery = `
WITH available AS (
  SELECT USER_ID, SUM(VALUE) AS TOTAL
  FROM (
    SELECT USER_ID, ACCRUAL AS VALUE FROM LS_ORDERS WHERE USER_ID = $2
  	UNION ALL
    SELECT USER_ID, -AMOUNT AS VALUE FROM LS_WITHDRAWALS WHERE USER_ID = $2
  )
  GROUP BY USER_ID
),
possible AS (
  SELECT USER_ID FROM available WHERE TOTAL >= $3
)
INSERT INTO LS_WITHDRAWALS (ID, USER_ID, AMOUNT)
SELECT $1, USER_ID, $3 FROM possible
RETURNING ID
`
	rows, err = r.db.Pool().Query(ctx, sqlQuery, wdId, userId, amount)
	if err != nil {
		return fmt.Errorf("error executing withdrawal query: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		return ErrInsufficientBalance
	}
	return nil
}

func (r *WithdrawalRepository) fromRow(rows pgx.Rows) (*model.Withdrawal, error) {
	if !rows.Next() {
		return nil, ErrEntityNotFound
	}
	return r.scan(rows)
}

func (r *WithdrawalRepository) fromRows(rows pgx.Rows) ([]*model.Withdrawal, error) {
	wds := make([]*model.Withdrawal, 0)
	for rows.Next() {
		wd, err := r.scan(rows)
		if err != nil {
			return nil, err
		}
		wds = append(wds, wd)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error reading rows from table %s: %w", r.tableName, err)
	}

	return wds, nil
}

func (r *WithdrawalRepository) scan(rows pgx.Rows) (*model.Withdrawal, error) {
	wd := model.Withdrawal{}
	err := rows.Scan(&wd.ID, &wd.UserID, &wd.Sum, &wd.ProcessedAt)
	if err != nil {
		return nil, err
	}
	return &wd, nil
}
