package model

import (
	"time"

	"github.com/google/uuid"
)

// User is main actor entity which is used by both authentication and loyalty data
// TODO: maybe we don't need this as a separate entity since there is no advanced user management right now
type User struct {
	Id          uuid.UUID `json:"id"`
	Login       string    `json:"login"`
	AuthHash    string    `json:"authHash"`
	CreatedAt   time.Time `json:"createdAt"`
	LastLoginAt time.Time `json:"lastLoginAt"`
}
