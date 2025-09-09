package model

type WithdrawOrder struct {
	OrderID string  `json:"orderID"`
	Sum     float64 `json:"sum"`
}
