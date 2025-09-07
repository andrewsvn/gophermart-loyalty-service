package db

import (
	"embed"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"go.uber.org/zap"

	_ "github.com/golang-migrate/migrate/v4/database/postgres"
)

//go:embed migration/*.sql
var migrationFS embed.FS

func Migrate(dbConnString string, logger *zap.Logger) error {
	fs, err := iofs.New(migrationFS, "migration")
	if err != nil {
		return fmt.Errorf("can't find migration files: %w", err)
	}
	m, err := migrate.NewWithSourceInstance("iofs", fs, dbConnString)
	if err != nil {
		return fmt.Errorf("can't initialize database migration: %w", err)
	}

	err = m.Up()
	if err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			logger.Sugar().Info("database schema is up to date")
		} else {
			return err
		}
	}
	return nil
}
