package model

import (
	"time"

	"github.com/google/uuid"
)

type Withdrawal struct {
	ID          string    `json:"order"`
	UserID      uuid.UUID `json:"-"`
	Sum         float64   `json:"sum"`
	ProcessedAt time.Time `json:"processedAt"`
}
