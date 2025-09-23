package postgres

import (
	"github.com/andrewsvn/gophermart-ls/internal/db"
	"github.com/andrewsvn/gophermart-ls/internal/repository"
)

type PgStorageManager struct {
	pgdb    *db.PostgresDB
	users   repository.UserStorage
	loyalty repository.LoyaltyStorage
}

func NewPgStorageManager(pgdb *db.PostgresDB) *PgStorageManager {
	return &PgStorageManager{
		pgdb:    pgdb,
		users:   NewUserPgStorage(pgdb),
		loyalty: NewLoyaltyPgStorage(pgdb),
	}
}

func (m *PgStorageManager) GetUserStorage() repository.UserStorage {
	return m.users
}

func (m *PgStorageManager) GetLoyaltyStorage() repository.LoyaltyStorage {
	return m.loyalty
}

func (m *PgStorageManager) Close() {
	m.pgdb.Close()
}
