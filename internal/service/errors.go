package service

import "errors"

var (
	ErrUserAlreadyExists  = errors.New("user already exists")
	ErrWrongLoginPassword = errors.New("wrong combination of login and password used")

	ErrInvalidOrderID          = errors.New("order ID has incorrect format")
	ErrWithdrawalAlreadyExists = errors.New("withdrawal already exists")
	ErrOrderExistsForSameUser  = errors.New("order already exists for the same user")
	ErrOrderExistsForOtherUser = errors.New("order already exists for another user")
	ErrNotEnoughBalance        = errors.New("not enough loyalty points available")
)
