package db

import (
	"context"
	"fmt"
	"strings"

	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresDB struct {
	dbpool *pgxpool.Pool
}

func NewPostgresDB(ctx context.Context, dsn string) (*PostgresDB, error) {
	dbc, err := pgxpool.New(ctx, strings.Trim(dsn, "\""))
	if err != nil {
		return nil, fmt.Errorf("failed to create database connection: %w", err)
	}

	return &PostgresDB{
		dbpool: dbc,
	}, nil
}

func (pgdb *PostgresDB) Close() {
	if pgdb.dbpool != nil {
		pgdb.dbpool.Close()
	}
}

func (pgdb *PostgresDB) Pool() *pgxpool.Pool {
	return pgdb.dbpool
}

func (pgdb *PostgresDB) Sqrl() squirrel.StatementBuilderType {
	return squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)
}
