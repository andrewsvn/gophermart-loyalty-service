package db

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrCreateTx = errors.New("error creating DB transaction")
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

func (pgdb *PostgresDB) BeginTx(ctx context.Context) (pgx.Tx, error) {
	tx, err := pgdb.dbpool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCreateTx, err)
	}
	return tx, nil
}

func (pgdb *PostgresDB) CommitTx(ctx context.Context, tx pgx.Tx) {
	_ = tx.Commit(ctx)
}

func (pgdb *PostgresDB) RollbackTx(ctx context.Context, tx pgx.Tx) {
	_ = tx.Rollback(ctx)
}
