package facile

import (
	"crypto/rand"
	"encoding/hex"
)

// NewID returns a random, prefixed identifier for a resource.
func NewID() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return "fac_" + hex.EncodeToString(b)
}
