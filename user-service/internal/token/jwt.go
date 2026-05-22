// Package token provides JWT generation and validation utilities.
package token

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims holds the parsed payload from a JWT.
type Claims struct {
	UserID string
	Role   string
}

// TokenManager defines the contract for JWT generation and validation.
type TokenManager interface {
	GenerateAccessToken(userID, role string) (token string, expiresIn int, err error)
	GenerateRefreshToken(userID string) (string, error)
	ValidateAccessToken(token string) (*Claims, error)
	ValidateRefreshToken(token string) (*Claims, error)
}

const (
	accessTokenTTL  = 3600          // 1 hour in seconds
	refreshTokenTTL = 7 * 24 * 3600 // 7 days in seconds
)

// jwtClaims is the internal JWT claims structure.
type jwtClaims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role,omitempty"`
	jwt.RegisteredClaims
}

// JWTManager implements TokenManager using HMAC-SHA256 signed JWTs.
type JWTManager struct {
	secret []byte
}

// NewJWTManager creates a new JWTManager with the given signing secret.
func NewJWTManager(secret string) *JWTManager {
	return &JWTManager{secret: []byte(secret)}
}

// GenerateAccessToken signs a JWT containing user_id and role, expiring in 3600s.
func (m *JWTManager) GenerateAccessToken(userID, role string) (string, int, error) {
	now := time.Now()
	claims := jwtClaims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(accessTokenTTL * time.Second)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.secret)
	if err != nil {
		return "", 0, err
	}
	return signed, accessTokenTTL, nil
}

// GenerateRefreshToken signs a JWT containing user_id, expiring in 7 days.
func (m *JWTManager) GenerateRefreshToken(userID string) (string, error) {
	now := time.Now()
	claims := jwtClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(refreshTokenTTL * time.Second)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

// ValidateAccessToken parses and verifies an access token, returning its claims.
func (m *JWTManager) ValidateAccessToken(tokenStr string) (*Claims, error) {
	return m.parse(tokenStr)
}

// ValidateRefreshToken parses and verifies a refresh token, returning its claims.
func (m *JWTManager) ValidateRefreshToken(tokenStr string) (*Claims, error) {
	return m.parse(tokenStr)
}

// parse is the shared JWT parsing logic for both token types.
func (m *JWTManager) parse(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &jwtClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return m.secret, nil
	})
	if err != nil {
		return nil, err
	}
	c, ok := token.Claims.(*jwtClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token claims")
	}
	return &Claims{UserID: c.UserID, Role: c.Role}, nil
}
