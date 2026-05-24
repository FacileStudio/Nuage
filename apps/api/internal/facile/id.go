package facile

import (
	"crypto/rand"
	"encoding/hex"
)

func NewID() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return "fac_" + hex.EncodeToString(b)
}
