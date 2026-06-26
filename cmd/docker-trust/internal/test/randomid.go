package test

import (
	"crypto/rand"
	"encoding/hex"
)

// RandomID returns a unique, 64-character ID consisting of a-z, 0-9.
func RandomID() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err) // This shouldn't happen
	}
	return hex.EncodeToString(b)
}
