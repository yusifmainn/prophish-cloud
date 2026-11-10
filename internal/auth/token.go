package auth

import (
	"crypto/rand"
	"encoding/base64"
)

// NewToken returns an unguessable URL-safe token used for per-target tracking links,
// so target email addresses never appear in URLs.
func NewToken() string {
	b := make([]byte, 24)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}
