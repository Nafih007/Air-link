package server

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
)

const tokenBytes = 16 // 128-bit

// NewToken returns a 128-bit random token as 32 hex characters.
func NewToken() (string, error) {
	var b [tokenBytes]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	return hex.EncodeToString(b[:]), nil
}

// TokenValid reports whether got matches want using a constant-time compare.
// Empty want never matches.
func TokenValid(want, got string) bool {
	if want == "" || len(want) != len(got) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(want), []byte(got)) == 1
}
