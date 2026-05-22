package token_test

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"pgregory.net/rapid"

	"user-service/internal/token"
)

const testSecret = "test-secret-key-for-property-tests"

func newManager() *token.JWTManager {
	return token.NewJWTManager(testSecret)
}

// validRoles returns the set of valid role strings.
var validRoles = []string{"CUSTOMER", "COURIER", "ADMIN"}

// genUserID produces a non-empty alphanumeric user ID.
func genUserID(t *rapid.T) string {
	return rapid.StringMatching(`[a-zA-Z0-9]{1,36}`).Draw(t, "userID")
}

// genRole picks one of the three valid roles.
func genRole(t *rapid.T) string {
	return rapid.SampledFrom(validRoles).Draw(t, "role")
}

// genNonJWT produces a string that is not a valid three-part JWT.
func genNonJWT(t *rapid.T) string {
	s := rapid.StringMatching(`[a-zA-Z0-9+/=]{1,50}`).Draw(t, "nonJWT")
	// Ensure it has fewer than 2 dots so it can never be a valid JWT structure.
	s = strings.ReplaceAll(s, ".", "")
	return s
}

// Feature: user-service, Property 8: Token expiry is within expected bounds
// Validates: Requirements 2.4
func TestProperty8_AccessTokenExpiryWithinBounds(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		mgr := newManager()
		userID := genUserID(rt)
		role := genRole(rt)

		before := time.Now()
		tokenStr, expiresIn, err := mgr.GenerateAccessToken(userID, role)
		if err != nil {
			rt.Fatalf("GenerateAccessToken failed: %v", err)
		}

		// expiresIn must be 3600
		if expiresIn != 3600 {
			rt.Fatalf("expected expiresIn=3600, got %d", expiresIn)
		}

		// Decode without verification to inspect exp claim directly.
		parsed, _, err := new(jwt.Parser).ParseUnverified(tokenStr, jwt.MapClaims{})
		if err != nil {
			rt.Fatalf("ParseUnverified failed: %v", err)
		}
		claims, ok := parsed.Claims.(jwt.MapClaims)
		if !ok {
			rt.Fatal("unexpected claims type")
		}

		expFloat, ok := claims["exp"].(float64)
		if !ok {
			rt.Fatal("exp claim missing or wrong type")
		}
		expTime := time.Unix(int64(expFloat), 0)

		expectedExp := before.Add(3600 * time.Second)
		tolerance := 60 * time.Second

		diff := expTime.Sub(expectedExp)
		if diff < -tolerance || diff > tolerance {
			rt.Fatalf("exp %v is not within 60s of expected %v (diff=%v)", expTime, expectedExp, diff)
		}
	})
}

// Feature: user-service, Property 9: Valid refresh token always yields a new access token
// Validates: Requirements 3.1
func TestProperty9_ValidRefreshTokenYieldsClaims(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		mgr := newManager()
		userID := genUserID(rt)

		refreshToken, err := mgr.GenerateRefreshToken(userID)
		if err != nil {
			rt.Fatalf("GenerateRefreshToken failed: %v", err)
		}

		claims, err := mgr.ValidateRefreshToken(refreshToken)
		if err != nil {
			rt.Fatalf("ValidateRefreshToken failed: %v", err)
		}
		if claims == nil {
			rt.Fatal("expected non-nil claims")
		}
		if claims.UserID != userID {
			rt.Fatalf("expected UserID=%q, got %q", userID, claims.UserID)
		}
	})
}

// Feature: user-service, Property 10: Malformed refresh token is always rejected
// Validates: Requirements 3.3
func TestProperty10_MalformedRefreshTokenRejected(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		mgr := newManager()
		bad := genNonJWT(rt)

		_, err := mgr.ValidateRefreshToken(bad)
		if err == nil {
			rt.Fatalf("expected error for malformed token %q, got nil", bad)
		}
	})
}
