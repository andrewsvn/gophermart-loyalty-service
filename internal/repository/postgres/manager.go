package postgres

import (
	"github.com/andrewsvn/gophermart-ls/internal/db"
	"github.com/andrewsvn/gophermart-ls/internal/repository"
)

type PgStorageManager struct {
	users   repository.UserStorage
	loyalty repository.LoyaltyStorage
}

func NewPgStorageManager(pgdb *db.PostgresDB) *PgStorageManager {
	return &PgStorageManager{
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
