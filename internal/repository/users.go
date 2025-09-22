package repository

import (
	"context"

	"github.com/andrewsvn/gophermart-ls/internal/model"
	"github.com/google/uuid"
)

type UserStorage interface {
	CreateUser(ctx context.Context, login, authHash string) (*uuid.UUID, error)
	UpdateUserLoginTS(ctx context.Context, userID uuid.UUID) (bool, error)

	GetUserByLogin(ctx context.Context, login string) (*model.User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*model.User, error)

	CheckUserExistsByLogin(ctx context.Context, login string) (bool, error)
}
