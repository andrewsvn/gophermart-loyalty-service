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

func (r *OrderRepository) UpdateOrderAccrual(
	ctx context.Context,
	orderAccrual *model.OrderAccrual,
	timestamp time.Time,
) error {
	ok, err := r.updateRows(ctx, func(ub squirrel.UpdateBuilder) squirrel.UpdateBuilder {
		return ub.
			Set("STATUS", orderAccrual.Status).
			Set("ACCRUAL", orderAccrual.Accrual).
			Set("LAST_UPDATE_TS", timestamp).
			Set("PENDING", false).
			Where(squirrel.Eq{"ID": orderAccrual.OrderID})
	})
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("%w: orderId='%s'", ErrEntityNotFound, orderAccrual.OrderID)
	}
	return nil
}

func (r *OrderRepository) FetchOrderIDsForUpdate(ctx context.Context, limit uint64) ([]string, error) {
	filterSql, _, err := r.sqrl.
		Select("ID").
		From(r.tableName).
		Where(squirrel.Expr("(STATUS = 'NEW' OR STATUS = 'PROCESSING') AND PENDING <> true")).
		OrderBy("LAST_UPDATE_TS ASC").
		Limit(limit).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("error building filter query: %w", err)
	}

	sqlQuery, args, err := r.sqrl.
		Update(r.tableName).
		Set("PENDING", true).
		Where(squirrel.Expr("ID IN (" + filterSql + ")")).
		Suffix("RETURNING ID").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("error building update query: %w", err)
	}

	rows, err := r.db.Pool().Query(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("error executing update query on table %s: %w", r.tableName, err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("error scanning update result row: %w", err)
		}
		ids = append(ids, id)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error scanning update result: %w", err)
	}

	return ids, nil
}

func (r *OrderRepository) ResetPendingOrders(ctx context.Context) error {
	_, err := r.updateRows(ctx, func(ub squirrel.UpdateBuilder) squirrel.UpdateBuilder {
		return ub.
			Set("PENDING", false).
			Where(squirrel.Eq{"PENDING": true})
	})
	return err
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
