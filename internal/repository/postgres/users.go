package postgres

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

type UserPgStorage struct {
	basePgStorage
}

func NewUserPgStorage(db *db.PostgresDB) *UserPgStorage {
	return &UserPgStorage{
		basePgStorage: basePgStorage{
			pgdb: db,
		},
	}
}

func (r *UserPgStorage) GetUserByLogin(ctx context.Context, login string) (*model.User, error) {
	sqlQuery, args, err := sqrl.
		Select(userColumns).
		From(userTableName).
		Where(squirrel.Eq{"LOGIN": login}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidQuery, err)
	}

	rows, err := r.query(ctx, nil, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("%w %s: %v", ErrExecuteSelect, userTableName, err)
	}
	defer rows.Close()

	return r.userFromRow(rows)
}

func (r *UserPgStorage) GetUserByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	sqlQuery, args, err := sqrl.
		Select(userColumns).
		From(userTableName).
		Where(squirrel.Eq{"ID": id.String()}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidQuery, err)
	}

	rows, err := r.query(ctx, nil, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("%w %s: %v", ErrExecuteSelect, userTableName, err)
	}
	defer rows.Close()

	return r.userFromRow(rows)
}

func (r *UserPgStorage) CreateUser(ctx context.Context, login, authHash string) (*uuid.UUID, error) {
	newID := uuid.New()
	sqlQuery, args, err := sqrl.
		Insert(userTableName).
		Columns("ID, LOGIN, AUTH_HASH, CREATE_TS").
		Values(
			newID.String(),
			login,
			authHash,
			time.Now(),
		).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidQuery, err)
	}

	_, err = r.exec(ctx, nil, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("%w %s: %v", ErrExecuteInsert, userTableName, err)
	}

	return &newID, nil
}

func (r *UserPgStorage) UpdateUserLoginTS(ctx context.Context, userID uuid.UUID) (bool, error) {
	sqlQuery, args, err := sqrl.
		Update(userTableName).
		Set("LAST_LOGIN_TS", time.Now()).
		Where(squirrel.Eq{"ID": userID}).
		ToSql()
	if err != nil {
		return false, fmt.Errorf("%w: %v", ErrExecuteUpdate, err)
	}

	res, err := r.exec(ctx, nil, sqlQuery, args...)
	if err != nil {
		return false, fmt.Errorf("%w %s: %v", ErrExecuteUpdate, userTableName, err)
	}
	return res.RowsAffected() > 0, nil
}

func (r *UserPgStorage) CheckUserExistsByLogin(ctx context.Context, login string) (bool, error) {
	sqlQuery, args, err := sqrl.
		Select("ID").
		From(userTableName).
		Where(squirrel.Eq{"LOGIN": login}).
		ToSql()
	if err != nil {
		return false, fmt.Errorf("%w %s: %v", ErrExecuteSelect, userTableName, err)
	}

	rows, err := r.query(ctx, nil, sqlQuery, args...)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	return rows.Next(), nil
}

func (r *UserPgStorage) userFromRow(rows pgx.Rows) (*model.User, error) {
	if !rows.Next() {
		return nil, nil
	}

	var user model.User
	err := rows.Scan(&user.ID, &user.Login, &user.AuthHash, &user.CreatedAt, &user.LastLoginAt)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrScanningRow, err)
	}
	return &user, nil
}
