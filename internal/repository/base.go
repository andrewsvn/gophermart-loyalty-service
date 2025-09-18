package repository

import (
	"context"

	"github.com/andrewsvn/gophermart-ls/internal/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type baseRepository struct {
	pgdb *db.PostgresDB
}

func (r *baseRepository) query(
	ctx context.Context, tx pgx.Tx, sqlQuery string, args ...interface{}) (pgx.Rows, error) {
	if tx == nil {
		return r.pgdb.Pool().Query(ctx, sqlQuery, args...)
	}
	return tx.Query(ctx, sqlQuery, args...)
}

func (r *baseRepository) exec(
	ctx context.Context, tx pgx.Tx, sqlQuery string, args ...interface{}) (pgconn.CommandTag, error) {
	if tx == nil {
		return r.pgdb.Pool().Exec(ctx, sqlQuery, args...)
	}
	return tx.Exec(ctx, sqlQuery, args...)
}
