package config

type AuthConfig struct {
	IdpKeyBase64 string `env:"IDENTITY_KEY"`
	// TODO: token lifecycle settings can go in there
}

func (cfg *AuthConfig) BindFlags() {
	// server secret must be set in environment parameter
}
