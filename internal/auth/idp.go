package auth

import (
	"encoding/base64"
	"fmt"
	"time"

	"github.com/andrewsvn/gophermart-ls/internal/config"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
)

type IdentityClaims struct {
	jwt.RegisteredClaims
	UserID       uuid.UUID
	UserAuthHash string
	Timestamp    time.Time
}

type IdentityProvider struct {
	cfg       *config.AuthConfig
	secretKey []byte
}

func NewIdentityProvider(cfg *config.AuthConfig) (*IdentityProvider, error) {
	secretKey, err := base64.StdEncoding.DecodeString(cfg.IdpKeyBase64)
	if err != nil {
		return nil, fmt.Errorf("server secret key can't be decoded: %v", err)
	}

	return &IdentityProvider{
		cfg:       cfg,
		secretKey: secretKey,
	}, nil
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
	return signedToken, nil
}

func (idp *IdentityProvider) ParseAccessToken(accessToken string) (*IdentityClaims, error) {
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
	return nil, fmt.Errorf("invalid token")
}
