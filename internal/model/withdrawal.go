package model

import "time"

type Withdrawal struct {
	ID          string    `json:"order"`
	UserID      string    `json:"-"`
	Sum         int       `json:"sum"`
	ProcessedAt time.Time `json:"processedAt"`
}
