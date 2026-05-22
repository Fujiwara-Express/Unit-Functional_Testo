package tracking_test

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"

	"order-service/tracking"
)

// alphanumericRe matches the expected tracking number format.
var alphanumericRe = regexp.MustCompile(`^[A-Z0-9]+$`)

// TestGenerateTrackingNumber covers unit-level cases for the generator.
// Requirements: 12.1, 12.2, 12.3, 13.6
func TestGenerateTrackingNumber(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "returns non-empty string",
			run: func(t *testing.T) {
				n := tracking.GenerateTrackingNumber()
				assert.NotEmpty(t, n, "tracking number must not be empty")
			},
		},
		{
			name: "two consecutive calls return distinct values",
			run: func(t *testing.T) {
				n1 := tracking.GenerateTrackingNumber()
				n2 := tracking.GenerateTrackingNumber()
				assert.NotEqual(t, n1, n2, "consecutive tracking numbers must be distinct")
			},
		},
		{
			name: "output matches alphanumeric format ^[A-Z0-9]+$",
			run: func(t *testing.T) {
				n := tracking.GenerateTrackingNumber()
				require.NotEmpty(t, n)
				assert.Regexp(t, alphanumericRe, n, "tracking number must match ^[A-Z0-9]+$")
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, tc.run)
	}
}

// TestTrackingNumberUniquenessAndFormat is a property-based test for Property 11.
//
// Property 11: For any set of independently generated tracking numbers, all values
// are distinct, non-empty, and match ^[A-Z0-9]+$.
//
// Validates: Requirements 12.1, 12.2, 12.3
func TestTrackingNumberUniquenessAndFormat(t *testing.T) {
	seen := make(map[string]struct{})

	rapid.Check(t, func(t *rapid.T) {
		n := tracking.GenerateTrackingNumber()

		require.NotEmpty(t, n, "tracking number must not be empty")
		require.Regexp(t, alphanumericRe, n, "tracking number must match ^[A-Z0-9]+$")

		_, exists := seen[n]
		require.False(t, exists, "duplicate tracking number detected: %s", n)

		seen[n] = struct{}{}
	})
}
