package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// SessionKeyFromToken returns a non-reversible stable key derived from a session token.
func SessionKeyFromToken(token string) string {
	token = strings.TrimSpace(token)
	if token == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
