package repository

import (
	"context"

	"github.com/Masterminds/squirrel"
	"github.com/andrewsvn/gophermart-ls/internal/db"
	"github.com/andrewsvn/gophermart-ls/internal/model"
	"github.com/google/uuid"
)

type WithdrawalRepository struct {
	baseRepository
}

const (
	withdrawalTableName = "LS_WITHDRAWALS"
	withdrawalColumns   = "ID, USER_ID, AMOUNT, CREATE_TS"
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
	// TODO
	return nil, nil
}

func (r *WithdrawalRepository) GetWithdrawalsByUserID(
	ctx context.Context,
	userID uuid.UUID,
) ([]*model.Withdrawal, error) {
	// TODO
	return nil, nil
}

func (r *WithdrawalRepository) GetTotalWithdrawalsByUserId(
	ctx context.Context,
	userID uuid.UUID,
) (float64, error) {
	// TODO
	return 0, nil
}

func (r *WithdrawalRepository) CreateWithdrawal(
	ctx context.Context,
	wdId string,
	userId uuid.UUID,
	amount float64,
) error {
	// TODO
	return nil
}
