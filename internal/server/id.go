package server

import (
	"crypto/rand"
	"encoding/hex"
)

// newID returns a short random identifier for new records and rates.
func newID() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
