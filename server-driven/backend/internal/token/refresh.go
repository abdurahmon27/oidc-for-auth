package token

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
)

const RefreshTokenBytes = 32

func GenerateRefreshToken() (raw string, hash string, err error) {
	b := make([]byte, RefreshTokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", "", err
	}
	raw = hex.EncodeToString(b)
	hash = HashToken(raw)
	return raw, hash, nil
}

func HashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
