package repository

import "errors"

var (
	ErrInvalidQuery         = errors.New("query can't be compiled")
	ErrDatabaseNotAvailable = errors.New("database is not available")
	ErrEntityNotFound       = errors.New("entity not found")
	ErrDuplicateEntity      = errors.New("duplicate entity found")
)
