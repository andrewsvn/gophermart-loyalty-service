package model

import (
	"time"

	"github.com/google/uuid"
)

// User is main actor entity which is used by both authentication and loyalty data
type User struct {
	ID          uuid.UUID  `json:"id"`
	Login       string     `json:"login"`
	AuthHash    string     `json:"authHash"`
	CreatedAt   *time.Time `json:"createdAt"`
	LastLoginAt *time.Time `json:"lastLoginAt"`
}
