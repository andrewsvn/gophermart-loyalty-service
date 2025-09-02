package repository

import (
	"context"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/andrewsvn/gophermart-ls/internal/db"
	"github.com/jackc/pgx/v5"
)

type baseRepository struct {
	db   *db.PostgresDB
	sqrl squirrel.StatementBuilderType

	columns   string
	tableName string
}

func (r *baseRepository) queryRows(
	ctx context.Context,
	selectCriteria func(squirrel.SelectBuilder) squirrel.SelectBuilder,
) (pgx.Rows, error) {
	sb := r.sqrl.Select(r.columns).From(r.tableName)
	sqlQuery, args, err := selectCriteria(sb).ToSql()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidQuery, err)
	}

	rows, err := r.db.Pool().Query(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDatabaseNotAvailable, err)
	}
	return rows, nil
}

func (r *baseRepository) insertRow(
	ctx context.Context,
	values ...interface{},
) error {
	sqlQuery, args, err := r.sqrl.Insert(r.tableName).Columns(r.columns).Values(values...).ToSql()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidQuery, err)
	}

	_, err = r.db.Pool().Exec(ctx, sqlQuery, args)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrDatabaseNotAvailable, err)
	}
	return nil
}

func (r *baseRepository) updateRow(
	ctx context.Context,
	updateClause func(squirrel.UpdateBuilder) squirrel.UpdateBuilder,
) (bool, error) {
	ub := r.sqrl.Update(r.tableName)
	sqlQuery, args, err := updateClause(ub).ToSql()
	if err != nil {
		return false, fmt.Errorf("%w: %v", ErrInvalidQuery, err)
	}

	res, err := r.db.Pool().Exec(ctx, sqlQuery, args...)
	if err != nil {
		return false, fmt.Errorf("%w: %v", ErrDatabaseNotAvailable, err)
	}
	return res.RowsAffected() > 0, nil
}
