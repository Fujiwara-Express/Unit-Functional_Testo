package service

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
)

// SignatureValidator validates HMAC-SHA512 signatures on callback requests.
type SignatureValidator struct {
	secretKey []byte
}

// NewSignatureValidator creates a new SignatureValidator with the given secret key.
func NewSignatureValidator(secretKey string) *SignatureValidator {
	return &SignatureValidator{secretKey: []byte(secretKey)}
}

// Compute returns the HMAC-SHA512 hex digest of the given payload.
func (v *SignatureValidator) Compute(payload string) string {
	mac := hmac.New(sha512.New, v.secretKey)
	mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}

// Validate returns true if the provided signature matches the HMAC-SHA512 of the payload.
func (v *SignatureValidator) Validate(payload, signature string) bool {
	expected := v.Compute(payload)
	return hmac.Equal([]byte(expected), []byte(signature))
}
