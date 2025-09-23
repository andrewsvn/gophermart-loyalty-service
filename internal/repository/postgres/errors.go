package postgres

import "errors"

var (
	ErrInvalidQuery  = errors.New("query can't be compiled")
	ErrExecuteSelect = errors.New("error selecting rows from table")
	ErrExecuteInsert = errors.New("error inserting row into table")
	ErrExecuteUpdate = errors.New("error updating row in table")
	ErrScanningRow   = errors.New("error scanning row")
	ErrFetchingRows  = errors.New("error fetching result rows")
)
