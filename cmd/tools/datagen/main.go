package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/andrewsvn/gophermart-ls/internal/config"
	"github.com/andrewsvn/gophermart-ls/internal/db"
	"github.com/andrewsvn/gophermart-ls/internal/logging"
	"github.com/andrewsvn/gophermart-ls/internal/model"
	"github.com/andrewsvn/gophermart-ls/internal/utils"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// datagen is a tool to generate some test data in gophermart database to be able to test endpoints without setup

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

const (
	nUsers        int   = 5
	nOrders       int   = 200
	nWithdrawals  int   = 10
	maxAccrual    int64 = 1000
	maxWithdrawal int64 = 2000
)

var userIDs []uuid.UUID
var userBalances []int64

var sqrl = squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)
var rnd = rand.New(rand.NewSource(time.Now().UnixNano()))

func run() error {
	cfg, err := config.GetDatagenConfig()
	if err != nil {
		return err
	}

	logger, err := logging.NewZapLogger(cfg.LogConfig)
	if err != nil {
		return err
	}

	logger.Info("migrating database schema")
	err = db.Migrate(cfg.DatabaseURL, logger)
	if err != nil {
		return err
	}

	ctx := context.Background()
	logger.Info("initializing storage")
	pgdb, err := db.NewPostgresDB(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pgdb.Close()

	// do everything in transaction
	tx, err := pgdb.Pool().Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	logger.Info("truncating tables")
	_, err = tx.Exec(ctx, "TRUNCATE TABLE LS_USERS")
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, "TRUNCATE TABLE LS_ORDERS")
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, "TRUNCATE TABLE LS_WITHDRAWALS")
	if err != nil {
		return err
	}

	if cfg.IsCleanup {
		logger.Info("cleanup mode - skipping data generation")
		return tx.Commit(ctx)
	}

	logger.Info("generating users")
	err = generateUsers(ctx, tx)
	if err != nil {
		return err
	}

	logger.Info("generating orders")
	err = generateOrders(ctx, tx)
	if err != nil {
		return err
	}

	logger.Info("generating withdrawals")
	err = generateWithdrawals(ctx, tx)
	if err != nil {
		return err
	}

	logger.Info("all test data generated")
	return tx.Commit(ctx)
}

func generateUsers(ctx context.Context, tx pgx.Tx) error {
	ib := sqrl.Insert("LS_USERS").Columns("ID", "LOGIN", "AUTH_HASH", "CREATE_TS")
	for i := 0; i < nUsers; i++ {
		id := uuid.New()
		userIDs = append(userIDs, id)
		userBalances = append(userBalances, 0)

		userLogin := fmt.Sprintf("user%d", i)
		userPass := fmt.Sprintf("pass%d", i)
		userAuth := utils.LoginPassHash(userLogin, userPass)
		ib = ib.Values(id, userLogin, userAuth, time.Now())
	}
	sql, args, err := ib.ToSql()
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, sql, args...)
	if err != nil {
		return err
	}

	return nil
}

func generateOrders(ctx context.Context, tx pgx.Tx) error {
	ib := sqrl.Insert("LS_ORDERS").Columns("ID", "USER_ID", "STATUS", "ACCRUAL", "CREATE_TS")
	for i := 0; i < nOrders; i++ {
		userRoll := rnd.Intn(nUsers)

		orderID := utils.GenerateLuhnNumber(rnd)
		userID := userIDs[userRoll]

		status := model.OrderStatusProcessed
		statusRoll := rnd.Intn(10)
		switch statusRoll {
		case 0:
			status = model.OrderStatusNew
		case 1:
			status = model.OrderStatusProcessing
		case 2:
			status = model.OrderStatusInvalid
		}

		var accrual int64
		if status == model.OrderStatusProcessed {
			accrual = rnd.Int63n(maxAccrual) + 1
			userBalances[userRoll] += accrual
		}

		ib = ib.Values(orderID, userID, status, accrual, time.Now())
	}
	sql, args, err := ib.ToSql()
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, sql, args...)
	if err != nil {
		return err
	}

	return nil
}

func generateWithdrawals(ctx context.Context, tx pgx.Tx) error {
	ib := sqrl.Insert("LS_WITHDRAWALS").Columns("ID", "USER_ID", "AMOUNT", "CREATE_TS")
	for i := 0; i < nWithdrawals; i++ {
		userRoll := rnd.Intn(nUsers)

		wdID := utils.GenerateLuhnNumber(rnd)
		userID := userIDs[userRoll]
		amount := rnd.Int63n(maxWithdrawal)
		ib = ib.Values(wdID, userID, amount, time.Now())
	}
	sql, args, err := ib.ToSql()
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, sql, args...)
	if err != nil {
		return err
	}

	return nil
}
