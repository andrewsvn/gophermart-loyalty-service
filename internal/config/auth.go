package config

type AuthConfig struct {
	IdpKeyBase64 string `env:"IDENTITY_KEY" required:"true"`
	// TODO: token lifecycle settings can go in there
}

func (cfg *AuthConfig) BindFlags() {
	// server secret must be set in environment parameter
}
