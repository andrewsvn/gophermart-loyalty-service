package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/andrewsvn/gophermart-ls/internal/model"
	"github.com/andrewsvn/gophermart-ls/internal/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (ls *LoyaltyPgStorage) GetOrderByID(ctx context.Context, orderID string) (*model.Order, error) {
	return ls.txGetOrderByID(ctx, nil, orderID)
}

func (ls *LoyaltyPgStorage) txGetOrderByID(ctx context.Context, tx pgx.Tx, orderID string) (*model.Order, error) {
	sqlQuery, args, err := sqrl.
		Select(orderColumns).
		From(orderTableName).
		Where(squirrel.Eq{"ID": orderID}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidQuery, err)
	}

	rows, err := ls.query(ctx, tx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrExecuteSelect, err)
	}
	defer rows.Close()

	return ls.orderFromRow(rows)
}

func (ls *LoyaltyPgStorage) GetOrdersByUserID(ctx context.Context, userID uuid.UUID) ([]*model.Order, error) {
	sqlQuery, args, err := sqrl.
		Select(orderColumns).
		From(orderTableName).
		Where(squirrel.Eq{"USER_ID": userID}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidQuery, err)
	}

	rows, err := ls.query(ctx, nil, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("%w %s: %v", ErrExecuteSelect, orderTableName, err)
	}
	defer rows.Close()

	return ls.ordersFromRows(rows)
}

func (ls *LoyaltyPgStorage) CreateNewOrder(ctx context.Context, orderID string, userID uuid.UUID) error {
	ts := time.Now()
	sqlQuery, args, err := sqrl.
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

	res, err := ls.exec(ctx, nil, sqlQuery, args...)
	if err != nil {
		return fmt.Errorf("%w %s: %v", ErrExecuteInsert, orderTableName, err)
	}

	if res.RowsAffected() == 0 {
		return fmt.Errorf("%w %s: nothing was inserted", ErrExecuteInsert, orderTableName)
	}
	return nil
}

func (ls *LoyaltyPgStorage) txSetOrderAccrual(
	ctx context.Context,
	tx pgx.Tx,
	orderAccrual *model.OrderAccrual,
	timestamp time.Time,
) error {
	sqlQuery, args, err := sqrl.
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

	res, err := ls.exec(ctx, tx, sqlQuery, args...)
	if err != nil {
		return fmt.Errorf("%w %s: %v", ErrExecuteUpdate, orderTableName, err)
	}
	if res.RowsAffected() == 0 {
		return fmt.Errorf("%w: orderId='%s'", repository.ErrEntityNotFound, orderAccrual.OrderID)
	}
	return nil
}

func (ls *LoyaltyPgStorage) txFetchAccruedTotal(ctx context.Context, tx pgx.Tx, userID uuid.UUID) (float64, error) {
	sqlQuery, args, err := sqrl.
		Select("COALESCE(SUM(ACCRUAL), 0)").
		From(orderTableName).
		Where(squirrel.Eq{"USER_ID": userID}).
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrInvalidQuery, err)
	}

	rows, err := ls.query(ctx, tx, sqlQuery, args...)
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

func (ls *LoyaltyPgStorage) FetchOrderIDsForUpdate(ctx context.Context, limit uint64) ([]string, error) {
	filterQuery, _, err := sqrl.
		Select("ID").
		From(orderTableName).
		Where(squirrel.Expr("(STATUS = 'NEW' OR STATUS = 'PROCESSING') AND PENDING <> true")).
		OrderBy("LAST_UPDATE_TS ASC").
		Limit(limit).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidQuery, err)
	}

	sqlQuery, args, err := sqrl.
		Update(orderTableName).
		Set("PENDING", true).
		Where(squirrel.Expr("ID IN (" + filterQuery + ")")).
		Suffix("RETURNING ID").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidQuery, err)
	}

	rows, err := ls.query(ctx, nil, sqlQuery, args...)
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

func (ls *LoyaltyPgStorage) ResetPendingOrders(ctx context.Context) error {
	sqlQuery, args, err := sqrl.
		Update(orderTableName).
		Set("PENDING", false).
		Where(squirrel.Eq{"PENDING": true}).
		ToSql()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidQuery, err)
	}

	_, err = ls.exec(ctx, nil, sqlQuery, args...)
	if err != nil {
		return fmt.Errorf("%w %s: %v", ErrExecuteUpdate, orderTableName, err)
	}
	return nil
}

func (ls *LoyaltyPgStorage) orderFromRow(rows pgx.Rows) (*model.Order, error) {
	if !rows.Next() {
		return nil, nil
	}
	return ls.scanOrder(rows)
}

func (ls *LoyaltyPgStorage) ordersFromRows(rows pgx.Rows) ([]*model.Order, error) {
	orders := make([]*model.Order, 0)
	for rows.Next() {
		order, err := ls.scanOrder(rows)
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

func (ls *LoyaltyPgStorage) scanOrder(rows pgx.Rows) (*model.Order, error) {
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
