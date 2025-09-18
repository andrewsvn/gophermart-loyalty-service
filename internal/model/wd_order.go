package model

type WithdrawOrder struct {
	OrderID string  `json:"order"`
	Sum     float64 `json:"sum"`
}
