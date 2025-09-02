package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/andrewsvn/gophermart-ls/internal/db"
	"github.com/andrewsvn/gophermart-ls/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type UserRepository struct {
	baseRepository
}

const (
	userTableName = "LS_USERS"
	userColumns   = "ID, LOGIN, AUTH_HASH, CREATE_TS, LAST_LOGIN_TS"
)

func NewUserRepository(db *db.PostgresDB) *UserRepository {
	return &UserRepository{
		baseRepository: baseRepository{
			db:        db,
			sqrl:      squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
			tableName: userTableName,
			columns:   userColumns,
		},
	}
}

func (r *UserRepository) GetUserByLogin(ctx context.Context, login string) (*model.User, error) {
	rows, err := r.queryRows(ctx, func(sb squirrel.SelectBuilder) squirrel.SelectBuilder {
		return sb.Where(squirrel.Eq{"LOGIN": login})
	})
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.fromRow(rows)
}

func (r *UserRepository) GetUserByID(ctx context.Context, id string) (*model.User, error) {
	rows, err := r.queryRows(ctx, func(sb squirrel.SelectBuilder) squirrel.SelectBuilder {
		return sb.Where(squirrel.Eq{"ID": id})
	})
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.fromRow(rows)
}

func (r *UserRepository) CreateUser(ctx context.Context, login, authHash string) (*uuid.UUID, error) {
	exists, err := r.userExists(ctx, login)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("%w: login='%s'", ErrDuplicateEntity, login)
	}

	newId := uuid.New()
	err = r.insertRow(ctx,
		newId,
		login,
		authHash,
		time.Now(),
		nil,
	)
	if err != nil {
		return nil, err
	}
	return &newId, nil
}

func (r *UserRepository) UpdateUserLoginTs(ctx context.Context, userId uuid.UUID) error {
	ok, err := r.updateRow(ctx, func(ub squirrel.UpdateBuilder) squirrel.UpdateBuilder {
		return ub.
			Set("LAST_LOGIN_TS", time.Now()).
			Where(squirrel.Eq{"ID": userId})
	})
	if err != nil {
		return err
	}
	if !ok {
		return ErrEntityNotFound
	}
	return nil
}

func (r *UserRepository) userExists(ctx context.Context, login string) (bool, error) {
	rows, err := r.queryRows(ctx, func(sb squirrel.SelectBuilder) squirrel.SelectBuilder {
		return sb.Where(squirrel.Eq{"LOGIN": login})
	})
	if err != nil {
		return false, err
	}
	defer rows.Close()
	return rows.Next(), nil
}

func (r *UserRepository) fromRow(rows pgx.Rows) (*model.User, error) {
	if !rows.Next() {
		return nil, ErrEntityNotFound
	}

	var user model.User
	err := rows.Scan(&user.Id, &user.Login, &user.AuthHash, &user.CreatedAt, &user.LastLoginAt)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDatabaseNotAvailable, err)
	}
	return &user, nil
}
