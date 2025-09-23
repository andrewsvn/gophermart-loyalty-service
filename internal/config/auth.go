package config

import (
	"encoding/base64"
	"fmt"
)

type AuthConfig struct {
	IdpKeyBase64 string `env:"IDENTITY_KEY"`
}

func (cfg *AuthConfig) BindFlags() {
	// server secret must be set in environment parameter
}

func (cfg *AuthConfig) Validate() error {
	_, err := base64.StdEncoding.DecodeString(cfg.IdpKeyBase64)
	if err != nil {
		return fmt.Errorf("server secret key can't be decoded: %v", err)
	}
	return nil
}
