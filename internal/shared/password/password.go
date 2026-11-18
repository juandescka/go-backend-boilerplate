// Package password implements Mapatu's bounded Argon2id PHC password format.
package password

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"strings"
	"unicode/utf8"

	"golang.org/x/crypto/argon2"
)

const (
	MaxBytes    = 1024
	memory      = 64 * 1024 // KiB: 64 MiB per verification, not per lane.
	iterations  = 3
	parallelism = 4
	saltBytes   = 16
	keyBytes    = 32
	phcPrefix   = "$argon2id$v=19$m=65536,t=3,p=4$"
)

func ValidInput(value string) bool {
	return len(value) > 0 && len(value) <= MaxBytes && utf8.ValidString(value)
}

// Hash is for internal writers/fixtures, not an HTTP registration endpoint.
func Hash(value string) (string, error) {
	if !ValidInput(value) {
		return "", errors.New("invalid password input")
	}
	salt := make([]byte, saltBytes)
	if _, err := rand.Read(salt); err != nil {
		return "", errors.New("password salt generation failed")
	}
	key := argon2.IDKey([]byte(value), salt, iterations, memory, parallelism, keyBytes)
	return phcPrefix + base64.RawStdEncoding.EncodeToString(salt) + "$" + base64.RawStdEncoding.EncodeToString(key), nil
}

// Verify accepts only the supported PHC policy. Missing/malformed hashes still
// perform one bounded KDF, avoiding a cheap unknown/no-password account path.
// Callers must bound concurrent verifications; Argon2 cannot be interrupted.
func Verify(encoded, value string) bool {
	if !ValidInput(value) {
		return false
	}
	salt, expected, valid := decode(encoded)
	if !valid {
		salt, expected = make([]byte, saltBytes), make([]byte, keyBytes)
	}
	actual := argon2.IDKey([]byte(value), salt, iterations, memory, parallelism, keyBytes)
	matched := subtle.ConstantTimeCompare(actual, expected) == 1
	return valid && matched
}

func decode(encoded string) ([]byte, []byte, bool) {
	if len(encoded) > 200 || !strings.HasPrefix(encoded, phcPrefix) {
		return nil, nil, false
	}
	parts := strings.Split(strings.TrimPrefix(encoded, phcPrefix), "$")
	if len(parts) != 2 {
		return nil, nil, false
	}
	salt, err := base64.RawStdEncoding.Strict().DecodeString(parts[0])
	if err != nil || len(salt) != saltBytes || base64.RawStdEncoding.EncodeToString(salt) != parts[0] {
		return nil, nil, false
	}
	key, err := base64.RawStdEncoding.Strict().DecodeString(parts[1])
	if err != nil || len(key) != keyBytes || base64.RawStdEncoding.EncodeToString(key) != parts[1] {
		return nil, nil, false
	}
	return salt, key, true
}
