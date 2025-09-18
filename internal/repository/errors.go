package repository

import "errors"

var (
	ErrInvalidQuery        = errors.New("query can't be compiled")
	ErrExecuteSelect       = errors.New("error selecting rows from table")
	ErrExecuteInsert       = errors.New("error inserting row into table")
	ErrExecuteUpdate       = errors.New("error updating row into table")
	ErrScanningRow         = errors.New("error scanning row")
	ErrFetchingRows        = errors.New("error fetching result rows")
	ErrEntityNotFound      = errors.New("entity not found")
	ErrDuplicateEntity     = errors.New("duplicate entity found")
	ErrInsufficientBalance = errors.New("insufficient balance")
)
