package utils

import (
	"net/http"

	"github.com/google/uuid"
)

type AuthUserKey string

const (
	AuthorizedUserIDVar AuthUserKey = "userID"
)

func WithUserID(
	handler func(rw http.ResponseWriter, r *http.Request, userID uuid.UUID),
) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		userID, ok := getUserID(r)
		if !ok {
			http.Error(rw, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		handler(rw, r, userID)
	}
}

func getUserID(r *http.Request) (uuid.UUID, bool) {
	ctx := r.Context()
	userID, ok := ctx.Value(AuthorizedUserIDVar).(uuid.UUID)
	if !ok {
		return uuid.Nil, false
	}
	return userID, true
}
