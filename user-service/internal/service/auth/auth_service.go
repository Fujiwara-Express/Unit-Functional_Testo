// Package auth contains the authentication service implementation.
package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"

	"user-service/internal/domain"
	"user-service/internal/repository"
	"user-service/internal/token"
)

// RegisterRequest holds the input for account registration.
type RegisterRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

// RegisterResponse is returned on successful registration.
type RegisterResponse struct {
	UserID string `json:"user_id"`
	Status string `json:"status"`
}

// LoginRequest holds credentials for authentication.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// TokenPairResponse is returned on successful login.
type TokenPairResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

// AccessTokenResponse is returned on a successful token refresh.
type AccessTokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

// AuthService defines the authentication business-logic contract.
type AuthService interface {
	Register(ctx context.Context, req RegisterRequest) (RegisterResponse, error)
	Login(ctx context.Context, req LoginRequest) (TokenPairResponse, error)
	RefreshToken(ctx context.Context, refreshToken string) (AccessTokenResponse, error)
	Logout(ctx context.Context, refreshToken string) error
}

// service is the concrete implementation of AuthService.
type service struct {
	users      repository.UserRepository
	tokens     repository.RefreshTokenRepository
	tokenMgr   token.TokenManager
	bcryptCost int
}

// New creates a new AuthService implementation using bcrypt.DefaultCost.
func New(
	users repository.UserRepository,
	tokens repository.RefreshTokenRepository,
	tokenMgr token.TokenManager,
) AuthService {
	return &service{
		users:      users,
		tokens:     tokens,
		tokenMgr:   tokenMgr,
		bcryptCost: bcrypt.DefaultCost,
	}
}

// NewWithCost creates a new AuthService with a configurable bcrypt cost (useful for testing).
func NewWithCost(
	users repository.UserRepository,
	tokens repository.RefreshTokenRepository,
	tokenMgr token.TokenManager,
	cost int,
) AuthService {
	return &service{
		users:      users,
		tokens:     tokens,
		tokenMgr:   tokenMgr,
		bcryptCost: cost,
	}
}

// Register validates inputs, hashes the password, persists the user, and returns the new user_id.
func (s *service) Register(ctx context.Context, req RegisterRequest) (RegisterResponse, error) {
	// Validate email
	if err := domain.ValidateEmail(req.Email); err != nil {
		return RegisterResponse{}, err
	}
	// Validate password length
	if err := domain.ValidatePassword(req.Password); err != nil {
		return RegisterResponse{}, err
	}
	// Validate role
	if err := domain.ValidateRole(domain.Role(req.Role)); err != nil {
		return RegisterResponse{}, err
	}
	// Validate phone (optional but must be valid format if provided)
	if err := domain.ValidatePhone(req.Phone); err != nil {
		return RegisterResponse{}, err
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), s.bcryptCost)
	if err != nil {
		return RegisterResponse{}, fmt.Errorf("auth.Register: hash password: %w", err)
	}

	user := domain.User{
		Name:         req.Name,
		Email:        req.Email,
		Phone:        req.Phone,
		PasswordHash: string(hash),
		Role:         domain.Role(req.Role),
		IsActive:     true,
	}

	created, err := s.users.Create(ctx, user)
	if err != nil {
		return RegisterResponse{}, err
	}

	return RegisterResponse{UserID: created.ID, Status: "CREATED"}, nil
}

// Login verifies credentials, checks account status, and issues a token pair.
func (s *service) Login(ctx context.Context, req LoginRequest) (TokenPairResponse, error) {
	user, err := s.users.FindByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return TokenPairResponse{}, domain.ErrUnauthorized
		}
		return TokenPairResponse{}, fmt.Errorf("auth.Login: find user: %w", err)
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return TokenPairResponse{}, domain.ErrUnauthorized
	}

	// Check account is active (Requirement 6.3)
	if !user.IsActive {
		return TokenPairResponse{}, domain.ErrForbidden
	}

	// Generate token pair
	accessToken, expiresIn, err := s.tokenMgr.GenerateAccessToken(user.ID, string(user.Role))
	if err != nil {
		return TokenPairResponse{}, fmt.Errorf("auth.Login: generate access token: %w", err)
	}

	refreshToken, err := s.tokenMgr.GenerateRefreshToken(user.ID)
	if err != nil {
		return TokenPairResponse{}, fmt.Errorf("auth.Login: generate refresh token: %w", err)
	}

	// Persist refresh token hash
	tokenHash := hashToken(refreshToken)
	rt := domain.RefreshToken{
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}
	if err := s.tokens.Save(ctx, rt); err != nil {
		return TokenPairResponse{}, fmt.Errorf("auth.Login: save refresh token: %w", err)
	}

	return TokenPairResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    expiresIn,
	}, nil
}

// RefreshToken validates the refresh token, looks it up in the DB, and issues a new access token.
func (s *service) RefreshToken(ctx context.Context, refreshToken string) (AccessTokenResponse, error) {
	claims, err := s.tokenMgr.ValidateRefreshToken(refreshToken)
	if err != nil {
		return AccessTokenResponse{}, domain.ErrUnauthorized
	}

	// Look up token hash in DB to ensure it hasn't been revoked
	tokenHash := hashToken(refreshToken)
	_, err = s.tokens.FindByTokenHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return AccessTokenResponse{}, domain.ErrUnauthorized
		}
		return AccessTokenResponse{}, fmt.Errorf("auth.RefreshToken: find token: %w", err)
	}

	// Fetch user to get current role
	user, err := s.users.FindByID(ctx, claims.UserID)
	if err != nil {
		return AccessTokenResponse{}, fmt.Errorf("auth.RefreshToken: find user: %w", err)
	}

	accessToken, expiresIn, err := s.tokenMgr.GenerateAccessToken(user.ID, string(user.Role))
	if err != nil {
		return AccessTokenResponse{}, fmt.Errorf("auth.RefreshToken: generate access token: %w", err)
	}

	return AccessTokenResponse{AccessToken: accessToken, ExpiresIn: expiresIn}, nil
}

// Logout validates the refresh token and revokes it in the DB.
func (s *service) Logout(ctx context.Context, refreshToken string) error {
	_, err := s.tokenMgr.ValidateRefreshToken(refreshToken)
	if err != nil {
		return domain.ErrUnauthorized
	}

	tokenHash := hashToken(refreshToken)
	if err := s.tokens.Revoke(ctx, tokenHash); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.ErrUnauthorized
		}
		return fmt.Errorf("auth.Logout: revoke token: %w", err)
	}
	return nil
}

// hashToken returns a SHA-256 hex digest of the given token string.
func hashToken(t string) string {
	sum := sha256.Sum256([]byte(t))
	return hex.EncodeToString(sum[:])
}
