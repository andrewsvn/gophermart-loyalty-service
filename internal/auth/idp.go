package auth

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/andrewsvn/gophermart-ls/internal/config"
	"github.com/andrewsvn/gophermart-ls/internal/logging"
	"github.com/andrewsvn/gophermart-ls/internal/repository"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type IdentityClaims struct {
	jwt.RegisteredClaims
	UserID       uuid.UUID
	UserAuthHash string
	Timestamp    time.Time
}

type IdentityProvider struct {
	cfg         *config.AuthConfig
	userStorage repository.UserStorage
	secretKey   []byte

	logger *zap.SugaredLogger
}

var (
	ErrInvalidToken = errors.New("invalid token")
)

func NewIdentityProvider(
	cfg *config.AuthConfig,
	us repository.UserStorage,
	l *zap.Logger,
) *IdentityProvider {
	logger := logging.ComponentLogger(l, "identity-provider")

	var secretKey []byte
	var err error
	if cfg.IdpKeyBase64 != "" {
		secretKey, err = base64.StdEncoding.DecodeString(cfg.IdpKeyBase64)
		if err != nil {
			logger.Warnw("configured server secret key can't be decoded", "error", err)
		}
	}
	if secretKey == nil {
		logger.Warn("IDP secret key is not provided, falling back to default one (insecure)")
		secretKey = []byte("gopherMarKET")
	}

	return &IdentityProvider{
		cfg:         cfg,
		userStorage: us,
		secretKey:   secretKey,
		logger:      logger,
	}
}

func (idp *IdentityProvider) GenerateAccessToken(userID uuid.UUID, authHash string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, &IdentityClaims{
		RegisteredClaims: jwt.RegisteredClaims{},
		UserID:           userID,
		UserAuthHash:     authHash,
		Timestamp:        time.Now(),
	})

	signedToken, err := token.SignedString(idp.secretKey)
	if err != nil {
		return "", err
	}

	idp.logger.Debugw("generated new access token for user", "userID", userID)
	return signedToken, nil
}

func (idp *IdentityProvider) AuthorizeUser(ctx context.Context, accessToken string) (*uuid.UUID, error) {
	identityClaims, err := idp.parseAccessToken(accessToken)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	user, err := idp.userStorage.GetUserByID(ctx, identityClaims.UserID)
	if err != nil {
		if errors.Is(err, repository.ErrEntityNotFound) {
			return nil, ErrInvalidToken
		}
		return nil, err
	}
	if user == nil {
		return nil, ErrInvalidToken
	}

	if user.AuthHash != identityClaims.UserAuthHash {
		return nil, ErrInvalidToken
	}

	return &identityClaims.UserID, nil
}

func (idp *IdentityProvider) parseAccessToken(accessToken string) (*IdentityClaims, error) {
	claims := &IdentityClaims{}
	tokenWithClaims, err := jwt.ParseWithClaims(accessToken, claims, func(token *jwt.Token) (interface{}, error) {
		return idp.secretKey, nil
	})
	if err != nil {
		return nil, err
	}

	if _, ok := tokenWithClaims.Method.(*jwt.SigningMethodHMAC); !ok {
		return nil, fmt.Errorf("unexpected signing method: %v", tokenWithClaims.Header["alg"])
	}

	if claims, ok := tokenWithClaims.Claims.(*IdentityClaims); ok && tokenWithClaims.Valid {
		return claims, nil
	}
	return nil, fmt.Errorf("invalid token format")
}
