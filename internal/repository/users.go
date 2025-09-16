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

func (r *UserRepository) GetUserByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	rows, err := r.queryRows(ctx, func(sb squirrel.SelectBuilder) squirrel.SelectBuilder {
		return sb.Where(squirrel.Eq{"ID": id.String()})
	})
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.fromRow(rows)
}

func (r *UserRepository) CreateUser(ctx context.Context, login, authHash string) (*uuid.UUID, error) {
	newID := uuid.New()
	err := r.insertRow(ctx,
		newID.String(),
		login,
		authHash,
		time.Now(),
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("error inserting row into table %s: %w", r.tableName, err)
	}

	return &newID, nil
}

func (r *UserRepository) UpdateUserLoginTS(ctx context.Context, userID uuid.UUID) (bool, error) {
	return r.updateRows(ctx, func(ub squirrel.UpdateBuilder) squirrel.UpdateBuilder {
		return ub.
			Set("LAST_LOGIN_TS", time.Now()).
			Where(squirrel.Eq{"ID": userID})
	})
}

func (r *UserRepository) CheckUserExistsByLogin(ctx context.Context, login string) (bool, error) {
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
		return nil, nil
	}

	var user model.User
	err := rows.Scan(&user.ID, &user.Login, &user.AuthHash, &user.CreatedAt, &user.LastLoginAt)
	if err != nil {
		return nil, fmt.Errorf("error scanning row from table %s: %w", r.tableName, err)
	}
	return &user, nil
}
