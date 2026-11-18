package authentication

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
)

// Tokens contain exactly 256 random bits, encoded without header-unsafe padding.
func newToken() (string, string, error) {
	value := make([]byte, 32)
	if _, err := rand.Read(value); err != nil {
		return "", "", errors.New("session token generation failed")
	}
	token := base64.RawURLEncoding.EncodeToString(value)
	return token, TokenHash(token), nil
}

func ValidToken(token string) bool {
	if len(token) != 43 {
		return false
	}
	value, err := base64.RawURLEncoding.Strict().DecodeString(token)
	return err == nil && len(value) == 32 && base64.RawURLEncoding.EncodeToString(value) == token
}

// TokenHash is for high-entropy session tokens only, never passwords.
func TokenHash(token string) string {
	value := sha256.Sum256([]byte(token))
	return hex.EncodeToString(value[:])
}
