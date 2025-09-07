package repository

import "errors"

var (
	ErrInvalidQuery    = errors.New("query can't be compiled")
	ErrEntityNotFound  = errors.New("entity not found")
	ErrDuplicateEntity = errors.New("duplicate entity found")
)
