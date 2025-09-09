package utils

import (
	"crypto/sha256"
	"encoding/base64"
)

func LoginPassHashBytes(login, password string) []byte {
	sha := sha256.New()
	sha.Write([]byte(login))
	sha.Write([]byte("@"))
	sha.Write([]byte(password))
	return sha.Sum(nil)
}

func LoginPassHash(login, password string) string {
	return base64.StdEncoding.EncodeToString(LoginPassHashBytes(login, password))
}
