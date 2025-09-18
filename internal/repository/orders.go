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

type orderRepository struct {
	baseRepository
	pgdb *db.PostgresDB
}

func newOrderRepository(db *db.PostgresDB) *orderRepository {
	return &orderRepository{
		baseRepository: baseRepository{
			pgdb: db,
		},
		pgdb: db,
	}
}

func (r *orderRepository) getOrder(ctx context.Context, tx pgx.Tx, orderID string) (*model.Order, error) {
	sqlQuery, args, err := r.pgdb.Sqrl().
		Select(orderColumns).
		From(orderTableName).
		Where(squirrel.Eq{"ID": orderID}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidQuery, err)
	}

	rows, err := r.query(ctx, tx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrExecuteSelect, err)
	}
	defer rows.Close()

	return r.orderFromRow(rows)
}

func (r *orderRepository) getOrdersByUserID(ctx context.Context, userID uuid.UUID) ([]*model.Order, error) {
	sqlQuery, args, err := r.pgdb.Sqrl().
		Select(orderColumns).
		From(orderTableName).
		Where(squirrel.Eq{"USER_ID": userID}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidQuery, err)
	}

	rows, err := r.pgdb.Pool().Query(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("%w %s: %v", ErrExecuteSelect, orderTableName, err)
	}
	defer rows.Close()

	return r.ordersFromRows(rows)
}

func (r *orderRepository) createNewOrder(ctx context.Context, orderID string, userID uuid.UUID) error {
	ts := time.Now()
	sqlQuery, args, err := r.pgdb.Sqrl().
		Insert(orderTableName).
		Columns(
			"ID",
			"USER_ID",
			"STATUS",
			"ACCRUAL",
			"CREATE_TS",
			"LAST_UPDATE_TS",
		).
		Values(
			orderID,
			userID,
			model.OrderStatusNew,
			0.0,
			ts,
			ts,
		).
		ToSql()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidQuery, err)
	}

	res, err := r.pgdb.Pool().Exec(ctx, sqlQuery, args...)
	if err != nil {
		return fmt.Errorf("%w %s: %v", ErrExecuteInsert, orderTableName, err)
	}

	if res.RowsAffected() == 0 {
		return fmt.Errorf("%w %s: nothing was inserted", ErrExecuteInsert, orderTableName)
	}
	return nil
}

func (r *orderRepository) setOrderAccrual(
	ctx context.Context,
	tx pgx.Tx,
	orderAccrual *model.OrderAccrual,
	timestamp time.Time,
) error {
	sqlQuery, args, err := r.pgdb.Sqrl().
		Update(orderTableName).
		Set("STATUS", orderAccrual.Status).
		Set("ACCRUAL", orderAccrual.Accrual).
		Set("LAST_UPDATE_TS", timestamp).
		Set("PENDING", false).
		Where(squirrel.Eq{"ID": orderAccrual.OrderID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidQuery, err)
	}

	res, err := r.exec(ctx, tx, sqlQuery, args...)
	if err != nil {
		return fmt.Errorf("%w %s: %v", ErrExecuteUpdate, orderTableName, err)
	}
	if res.RowsAffected() == 0 {
		return fmt.Errorf("%w: orderId='%s'", ErrEntityNotFound, orderAccrual.OrderID)
	}
	return nil
}

func (r *orderRepository) fetchAccruedSum(ctx context.Context, tx pgx.Tx, userID uuid.UUID) (float64, error) {
	sqlQuery, args, err := r.pgdb.Sqrl().
		Select("COALESCE(SUM(ACCRUAL), 0)").
		From(orderTableName).
		Where(squirrel.Eq{"USER_ID": userID}).
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrInvalidQuery, err)
	}

	rows, err := r.query(ctx, tx, sqlQuery, args...)
	if err != nil {
		return 0, fmt.Errorf("%w %s: %w", ErrExecuteSelect, orderTableName, err)
	}
	defer rows.Close()

	if !rows.Next() {
		return 0, nil
	}

	var total float64
	if err := rows.Scan(&total); err != nil {
		return 0, fmt.Errorf("%w: %v", ErrScanningRow, err)
	}
	return total, nil
}

func (r *orderRepository) fetchOrderIDsForUpdate(ctx context.Context, limit uint64) ([]string, error) {
	filterQuery, _, err := r.pgdb.Sqrl().
		Select("ID").
		From(orderTableName).
		Where(squirrel.Expr("(STATUS = 'NEW' OR STATUS = 'PROCESSING') AND PENDING <> true")).
		OrderBy("LAST_UPDATE_TS ASC").
		Limit(limit).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidQuery, err)
	}

	sqlQuery, args, err := r.pgdb.Sqrl().
		Update(orderTableName).
		Set("PENDING", true).
		Where(squirrel.Expr("ID IN (" + filterQuery + ")")).
		Suffix("RETURNING ID").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidQuery, err)
	}

	rows, err := r.pgdb.Pool().Query(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("%w %s: %v", ErrExecuteUpdate, orderTableName, err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrScanningRow, err)
		}
		ids = append(ids, id)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrFetchingRows, err)
	}

	return ids, nil
}

func (r *orderRepository) resetPendingOrders(ctx context.Context) error {
	sqlQuery, args, err := r.pgdb.Sqrl().
		Update(orderTableName).
		Set("PENDING", false).
		Where(squirrel.Eq{"PENDING": true}).
		ToSql()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidQuery, err)
	}

	_, err = r.pgdb.Pool().Exec(ctx, sqlQuery, args...)
	if err != nil {
		return fmt.Errorf("%w %s: %v", ErrExecuteUpdate, orderTableName, err)
	}
	return nil
}

func (r *orderRepository) orderFromRow(rows pgx.Rows) (*model.Order, error) {
	if !rows.Next() {
		return nil, nil
	}
	return r.scanOrder(rows)
}

func (r *orderRepository) ordersFromRows(rows pgx.Rows) ([]*model.Order, error) {
	orders := make([]*model.Order, 0)
	for rows.Next() {
		order, err := r.scanOrder(rows)
		if err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrFetchingRows, err)
	}

	return orders, nil
}

func (r *orderRepository) scanOrder(rows pgx.Rows) (*model.Order, error) {
	var order model.Order
	var userIDStr string
	err := rows.Scan(&order.ID, &userIDStr, &order.Status, &order.Accrual, &order.UploadedAt, &order.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrScanningRow, err)
	}

	order.UserID, err = uuid.Parse(userIDStr)
	if err != nil {
		return nil, err
	}

	return &order, nil
}
