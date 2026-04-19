package auth

import (
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
)

const (
	passwordSaltBytes  = 16
	passwordKeyBytes   = 32
	passwordIterations = 60000
)

func HashPassword(password string) (salt string, hash string, iterations int, err error) {
	saltBytes := make([]byte, passwordSaltBytes)
	if _, err = rand.Read(saltBytes); err != nil {
		return "", "", 0, err
	}
	derived, err := pbkdf2.Key(sha256.New, password, saltBytes, passwordIterations, passwordKeyBytes)
	if err != nil {
		return "", "", 0, err
	}
	return base64.RawStdEncoding.EncodeToString(saltBytes), base64.RawStdEncoding.EncodeToString(derived), passwordIterations, nil
}

func VerifyPassword(password, salt, hash string, iterations int) bool {
	if password == "" || salt == "" || hash == "" || iterations <= 0 {
		return false
	}
	saltBytes, err := base64.RawStdEncoding.DecodeString(salt)
	if err != nil {
		return false
	}
	expected, err := base64.RawStdEncoding.DecodeString(hash)
	if err != nil {
		return false
	}
	derived, err := pbkdf2.Key(sha256.New, password, saltBytes, iterations, len(expected))
	if err != nil {
		return false
	}
	return subtle.ConstantTimeCompare(expected, derived) == 1
}
