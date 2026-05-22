package service_test

import (
	"testing"

	"payment-service/service"
)

const testSecretKey = "super-secret-key"

func TestSignatureValidator_ValidSignature(t *testing.T) {
	v := service.NewSignatureValidator(testSecretKey)
	payload := `{"order_id":"ORD-001","amount":50000,"status":"SUCCESS"}`
	sig := v.Compute(payload)

	if !v.Validate(payload, sig) {
		t.Error("expected valid signature to be accepted")
	}
}

func TestSignatureValidator_InvalidSignature(t *testing.T) {
	v := service.NewSignatureValidator(testSecretKey)
	payload := `{"order_id":"ORD-001","amount":50000,"status":"SUCCESS"}`

	if v.Validate(payload, "invalid-signature") {
		t.Error("expected invalid signature to be rejected")
	}
}

func TestSignatureValidator_MissingSignature(t *testing.T) {
	v := service.NewSignatureValidator(testSecretKey)
	payload := `{"order_id":"ORD-001"}`

	if v.Validate(payload, "") {
		t.Error("expected empty signature to be rejected")
	}
}

func TestSignatureValidator_Determinism(t *testing.T) {
	v := service.NewSignatureValidator(testSecretKey)
	payload := `{"order_id":"ORD-001","amount":50000}`

	sig1 := v.Compute(payload)
	sig2 := v.Compute(payload)

	if sig1 != sig2 {
		t.Errorf("expected deterministic output, got %q and %q", sig1, sig2)
	}
}

func TestSignatureValidator_RoundTrip(t *testing.T) {
	v := service.NewSignatureValidator(testSecretKey)
	payloads := []string{
		`{"order_id":"ORD-001","amount":50000,"status":"SUCCESS"}`,
		`{"order_id":"ORD-002","amount":0}`,
		`plain text payload`,
	}

	for _, payload := range payloads {
		sig := v.Compute(payload)
		if !v.Validate(payload, sig) {
			t.Errorf("round-trip failed for payload: %s", payload)
		}
	}
}
