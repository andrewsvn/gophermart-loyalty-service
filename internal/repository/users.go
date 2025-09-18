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

type userRepository struct {
	baseRepository
	pgdb *db.PostgresDB
}

func newUserRepository(db *db.PostgresDB) *userRepository {
	return &userRepository{
		baseRepository: baseRepository{
			pgdb: db,
		},
		pgdb: db,
	}
}

func (r *userRepository) getUserByLogin(ctx context.Context, login string) (*model.User, error) {
	sqlQuery, args, err := r.pgdb.Sqrl().
		Select(userColumns).
		From(userTableName).
		Where(squirrel.Eq{"LOGIN": login}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidQuery, err)
	}

	rows, err := r.pgdb.Pool().Query(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("%w %s: %v", ErrExecuteSelect, userTableName, err)
	}
	defer rows.Close()

	return r.userFromRow(rows)
}

func (r *userRepository) getUserByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	sqlQuery, args, err := r.pgdb.Sqrl().
		Select(userColumns).
		From(userTableName).
		Where(squirrel.Eq{"ID": id.String()}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidQuery, err)
	}

	rows, err := r.pgdb.Pool().Query(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("%w %s: %w", ErrExecuteSelect, userTableName, err)
	}
	defer rows.Close()

	return r.userFromRow(rows)
}

func (r *userRepository) createUser(ctx context.Context, login, authHash string) (*uuid.UUID, error) {
	newID := uuid.New()
	sqlQuery, args, err := r.pgdb.Sqrl().
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
		return nil, fmt.Errorf("%w: %w", ErrInvalidQuery, err)
	}

	_, err = r.exec(ctx, nil, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("%w %s: %w", ErrExecuteInsert, userTableName, err)
	}

	return &newID, nil
}

func (r *userRepository) updateUserLoginTS(ctx context.Context, userID uuid.UUID) (bool, error) {
	sqlQuery, args, err := r.pgdb.Sqrl().
		Update(userTableName).
		Set("LAST_LOGIN_TS", time.Now()).
		Where(squirrel.Eq{"ID": userID}).
		ToSql()
	if err != nil {
		return false, fmt.Errorf("%w: %w", ErrExecuteUpdate, err)
	}

	res, err := r.pgdb.Pool().Exec(ctx, sqlQuery, args...)
	if err != nil {
		return false, fmt.Errorf("%w %s: %w", ErrExecuteUpdate, userTableName, err)
	}
	return res.RowsAffected() > 0, nil
}

func (r *userRepository) checkUserExistsByLogin(ctx context.Context, login string) (bool, error) {
	sqlQuery, args, err := r.pgdb.Sqrl().
		Select("ID").
		From(userTableName).
		Where(squirrel.Eq{"LOGIN": login}).
		ToSql()
	if err != nil {
		return false, fmt.Errorf("%w: %w", ErrExecuteSelect, err)
	}

	rows, err := r.pgdb.Pool().Query(ctx, sqlQuery, args...)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	return rows.Next(), nil
}

func (r *userRepository) userFromRow(rows pgx.Rows) (*model.User, error) {
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
