package repository

const (
	userTableName       = "LS_USERS"
	userColumns         = "ID, LOGIN, AUTH_HASH, CREATE_TS, LAST_LOGIN_TS"
	orderTableName      = "LS_ORDERS"
	orderColumns        = "ID, USER_ID, STATUS, ACCRUAL, CREATE_TS, LAST_UPDATE_TS"
	withdrawalTableName = "LS_WITHDRAWALS"
	withdrawalColumns   = "ID, USER_ID, AMOUNT, CREATE_TS"
	balanceTableName    = "LS_BALANCES"
)
