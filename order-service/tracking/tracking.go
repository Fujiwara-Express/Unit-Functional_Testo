package tracking

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
)

// GenerateTrackingNumber produces a unique tracking number using random bytes.
// The result is an uppercase alphanumeric string matching ^[A-Z0-9]+$.
func GenerateTrackingNumber() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return strings.ToUpper(hex.EncodeToString(b))
}
