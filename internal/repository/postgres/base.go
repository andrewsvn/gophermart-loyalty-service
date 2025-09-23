package postgres

import (
	"context"

	"github.com/Masterminds/squirrel"
	"github.com/andrewsvn/gophermart-ls/internal/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var sqrl = squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)

type basePgStorage struct {
	pgdb *db.PostgresDB
}

func (bs *basePgStorage) db() *db.PostgresDB {
	return bs.pgdb
}

func (bs *basePgStorage) query(
	ctx context.Context, tx pgx.Tx, sqlQuery string, args ...interface{}) (pgx.Rows, error) {
	if tx == nil {
		return bs.pgdb.Pool().Query(ctx, sqlQuery, args...)
	}
	return tx.Query(ctx, sqlQuery, args...)
}

func (bs *basePgStorage) exec(
	ctx context.Context, tx pgx.Tx, sqlQuery string, args ...interface{}) (pgconn.CommandTag, error) {
	if tx == nil {
		return bs.pgdb.Pool().Exec(ctx, sqlQuery, args...)
	}
	return tx.Exec(ctx, sqlQuery, args...)
}
