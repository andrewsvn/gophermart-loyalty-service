package repository

import "errors"

var (
	ErrEntityNotFound      = errors.New("entity not found")
	ErrDuplicateEntity     = errors.New("duplicate entity found")
	ErrInsufficientBalance = errors.New("insufficient balance")
)
