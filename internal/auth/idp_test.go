package auth

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/andrewsvn/gophermart-ls/internal/config"
	"github.com/andrewsvn/gophermart-ls/internal/logging"
	"github.com/andrewsvn/gophermart-ls/internal/mocks"
	"github.com/andrewsvn/gophermart-ls/internal/model"
	"github.com/andrewsvn/gophermart-ls/internal/utils"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestIdentityProviderTokens(t *testing.T) {
	const nUsers = 3

	userIDs := make([]uuid.UUID, nUsers)
	userLogins := make([]string, nUsers)
	userPasswords := make([]string, nUsers)
	userHashes := make([]string, nUsers)
	dbUsers := make(map[uuid.UUID]*model.User, nUsers)

	ts := time.Now()
	for i := 0; i < nUsers; i++ {
		userIDs[i] = uuid.New()
		userLogins[i] = fmt.Sprintf("user%d", i)
		userPasswords[i] = fmt.Sprintf("pass%d", i)
		userHashes[i] = utils.LoginPassHash(userLogins[i], userPasswords[i])
		dbUsers[userIDs[i]] = &model.User{
			ID:        userIDs[i],
			Login:     userLogins[i],
			AuthHash:  userHashes[i],
			CreatedAt: &ts,
		}
	}

	userStor := new(mocks.MockUserStorage)
	for i := 0; i < nUsers; i++ {
		userStor.EXPECT().GetUserByID(mock.Anything, userIDs[i]).Return(dbUsers[userIDs[i]], nil)
	}
	userStor.EXPECT().GetUserByID(mock.Anything, mock.Anything).Return(nil, nil)

	l, err := logging.NewZapLogger(config.LogConfig{Level: "info"})
	require.NoError(t, err)

	idp := NewIdentityProvider(&config.AuthConfig{}, userStor, l)
	assert.NotEmpty(t, idp.secretKey)

	for i := 0; i < nUsers; i++ {
		token, err := idp.GenerateAccessToken(userIDs[i], userHashes[i])
		assert.NoError(t, err)
		assert.NotEmpty(t, token)

		claims, err := idp.parseAccessToken(token)
		assert.NoError(t, err)
		assert.Equal(t, userIDs[i], claims.UserID)
		assert.Equal(t, userHashes[i], claims.UserAuthHash)

		userID, err := idp.AuthorizeUser(context.Background(), token)
		assert.NoError(t, err)
		assert.NotNil(t, userID)
		assert.Equal(t, userIDs[i], *userID)
	}

	wrongPass := "abracadabra"
	wrongHash := utils.LoginPassHash(userLogins[0], wrongPass)

	token, err := idp.GenerateAccessToken(userIDs[0], wrongHash)
	assert.NoError(t, err)

	_, err = idp.AuthorizeUser(context.Background(), token)
	assert.EqualError(t, err, ErrInvalidToken.Error())
}
