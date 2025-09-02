package model

import "time"

type Withdrawal struct {
	ID        string    `json:"id"`
	UserID    string    `json:"userId"`
	Amount    int       `json:"amount"`
	CreatedAt time.Time `json:"createdAt"`
}
