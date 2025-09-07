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
	newId := uuid.New()
	sqlQuery, args, err := r.sqrl.
		Insert(r.tableName).
		Columns(
			"ID",
			"LOGIN",
			"AUTH_HASH",
			"CREATE_TS",
			"LAST_LOGIN_TS",
		).
		Values(
			newId.String(),
			login,
			authHash,
			time.Now(),
			nil,
		).
		ToSql()
	if err != nil {
		return nil, err
	}

	_, err = r.db.Pool().Exec(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("error inserting row into table %s: %w", r.tableName, err)
	}

	return &newId, nil
}

func (r *UserRepository) UpdateUserLoginTs(ctx context.Context, userId uuid.UUID) (bool, error) {
	return r.updateRow(ctx, func(ub squirrel.UpdateBuilder) squirrel.UpdateBuilder {
		return ub.
			Set("LAST_LOGIN_TS", time.Now()).
			Where(squirrel.Eq{"ID": userId})
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
