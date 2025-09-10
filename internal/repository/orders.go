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

type OrderRepository struct {
	baseRepository
}

const (
	orderTableName = "LS_ORDERS"
	orderColumns   = "ID, USER_ID, STATUS, ACCRUAL, CREATE_TS, LAST_UPDATE_TS"
)

func NewOrderRepository(db *db.PostgresDB) *OrderRepository {
	return &OrderRepository{
		baseRepository{
			db:        db,
			sqrl:      squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
			tableName: orderTableName,
			columns:   orderColumns,
		},
	}
}

func (r *OrderRepository) GetOrderByID(ctx context.Context, orderId string) (*model.Order, error) {
	rows, err := r.queryRows(ctx, func(sb squirrel.SelectBuilder) squirrel.SelectBuilder {
		return sb.Where(squirrel.Eq{"ID": orderId})
	})
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.fromRow(rows)
}

func (r *OrderRepository) GetOrdersByUserId(ctx context.Context, userId uuid.UUID) ([]*model.Order, error) {
	rows, err := r.queryRows(ctx, func(sb squirrel.SelectBuilder) squirrel.SelectBuilder {
		return sb.Where(squirrel.Eq{"USER_ID": userId}).OrderBy("ID ASC")
	})
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.fromRows(rows)
}

func (r *OrderRepository) GetTotalAccrualByUserId(ctx context.Context, userId uuid.UUID) (float64, error) {
	sqlQuery, args, err := r.sqrl.
		Select("COALESCE(SUM(ACCRUAL), 0)").
		From(r.tableName).
		Where(squirrel.Eq{"USER_ID": userId}).
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrInvalidQuery, err)
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
		return 0, fmt.Errorf("error scanning rows from table %s: %w", r.tableName, err)
	}
	return total, nil
}

func (r *OrderRepository) CreateNewOrder(ctx context.Context, orderId string, userId uuid.UUID) error {
	ts := time.Now()
	return r.insertRow(ctx,
		orderId,
		userId,
		model.OrderStatusNew,
		0.0,
		ts,
		ts,
	)
}

func (r *OrderRepository) UpdateOrder(
	ctx context.Context,
	orderId string,
	status model.OrderStatus,
	accrual float64,
) error {
	ok, err := r.updateRow(ctx, func(ub squirrel.UpdateBuilder) squirrel.UpdateBuilder {
		return ub.
			Set("STATUS", status).
			Set("ACCRUAL", accrual).
			Set("LAST_UPDATE_TS", time.Now()).
			Where(squirrel.Eq{"ID": orderId})
	})
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("%w: orderId='%s'", ErrEntityNotFound, orderId)
	}
	return nil
}

func (r *OrderRepository) fromRow(rows pgx.Rows) (*model.Order, error) {
	if !rows.Next() {
		return nil, nil
	}
	return r.scan(rows)
}

func (r *OrderRepository) fromRows(rows pgx.Rows) ([]*model.Order, error) {
	orders := make([]*model.Order, 0)
	for rows.Next() {
		order, err := r.scan(rows)
		if err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error reading rows from table %s: %w", r.tableName, err)
	}

	return orders, nil
}

func (r *OrderRepository) scan(rows pgx.Rows) (*model.Order, error) {
	var order model.Order
	err := rows.Scan(&order.ID, &order.UserID, &order.Status, &order.Accrual, &order.UploadedAt, &order.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &order, nil
}
