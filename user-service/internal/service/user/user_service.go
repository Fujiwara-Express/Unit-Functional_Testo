// Package user contains the user profile service implementation.
package user

import (
	"context"
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"

	"user-service/internal/domain"
	"user-service/internal/repository"
)

// UserProfile is the profile data returned to callers (no password hash).
type UserProfile struct {
	UserID    string `json:"user_id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	Phone     string `json:"phone"`
	Role      string `json:"role"`
	IsActive  bool   `json:"is_active"`
	CreatedAt string `json:"created_at"`
}

// UpdateProfileRequest holds the fields a user may update.
type UpdateProfileRequest struct {
	Name  string `json:"name"`
	Phone string `json:"phone"`
}

// UpdateProfileResponse is returned after a successful profile update.
type UpdateProfileResponse struct {
	UserID string `json:"user_id"`
	Status string `json:"status"`
}

// ChangePasswordRequest holds the current and new passwords for a password change.
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// UserService defines the user profile business-logic contract.
type UserService interface {
	GetProfile(ctx context.Context, userID string) (UserProfile, error)
	UpdateProfile(ctx context.Context, userID string, req UpdateProfileRequest) (UpdateProfileResponse, error)
	SetAccountStatus(ctx context.Context, callerRole string, targetID string, active bool) error
	ChangePassword(ctx context.Context, userID string, req ChangePasswordRequest) error
}

// service is the concrete implementation of UserService.
type service struct {
	users      repository.UserRepository
	bcryptCost int
}

// New creates a new UserService implementation.
func New(users repository.UserRepository) UserService {
	return &service{users: users, bcryptCost: bcrypt.DefaultCost}
}

// NewWithCost creates a new UserService with a configurable bcrypt cost (useful for testing).
func NewWithCost(users repository.UserRepository, cost int) UserService {
	return &service{users: users, bcryptCost: cost}
}

// GetProfile fetches a user by ID and returns the profile without the password hash.
// Requirements: 4.1, 8.1
func (s *service) GetProfile(ctx context.Context, userID string) (UserProfile, error) {
	u, err := s.users.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return UserProfile{}, domain.ErrNotFound
		}
		return UserProfile{}, fmt.Errorf("user.GetProfile: %w", err)
	}
	return toProfile(u), nil
}

// UpdateProfile validates name and phone, updates the user record, and returns status UPDATED.
// Requirements: 4.2, 4.3, 4.4
func (s *service) UpdateProfile(ctx context.Context, userID string, req UpdateProfileRequest) (UpdateProfileResponse, error) {
	// Validate phone if provided
	if err := domain.ValidatePhone(req.Phone); err != nil {
		return UpdateProfileResponse{}, err
	}

	u, err := s.users.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return UpdateProfileResponse{}, domain.ErrNotFound
		}
		return UpdateProfileResponse{}, fmt.Errorf("user.UpdateProfile: find user: %w", err)
	}

	if req.Name != "" {
		u.Name = req.Name
	}
	if req.Phone != "" {
		u.Phone = req.Phone
	}

	if _, err := s.users.Update(ctx, u); err != nil {
		return UpdateProfileResponse{}, fmt.Errorf("user.UpdateProfile: update: %w", err)
	}

	return UpdateProfileResponse{UserID: userID, Status: "UPDATED"}, nil
}

// SetAccountStatus verifies the caller is an admin, then sets the is_active flag.
// Requirements: 6.1, 6.2, 6.4
func (s *service) SetAccountStatus(ctx context.Context, callerRole string, targetID string, active bool) error {
	if domain.Role(callerRole) != domain.RoleAdmin {
		return domain.ErrForbidden
	}
	if err := s.users.SetStatus(ctx, targetID, active); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.ErrNotFound
		}
		return fmt.Errorf("user.SetAccountStatus: %w", err)
	}
	return nil
}

// ChangePassword verifies the current password and updates the hash if valid.
// Requirements: 5.1, 5.2, 5.3
func (s *service) ChangePassword(ctx context.Context, userID string, req ChangePasswordRequest) error {
	// Validate new password length
	if err := domain.ValidatePassword(req.NewPassword); err != nil {
		return err
	}

	u, err := s.users.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.ErrNotFound
		}
		return fmt.Errorf("user.ChangePassword: find user: %w", err)
	}

	// Verify current password
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.CurrentPassword)); err != nil {
		return domain.ErrUnauthorized
	}

	// Hash new password
	newHash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), s.bcryptCost)
	if err != nil {
		return fmt.Errorf("user.ChangePassword: hash password: %w", err)
	}

	u.PasswordHash = string(newHash)
	if _, err := s.users.Update(ctx, u); err != nil {
		return fmt.Errorf("user.ChangePassword: update: %w", err)
	}
	return nil
}

// toProfile converts a domain.User to a UserProfile (omitting password hash).
func toProfile(u domain.User) UserProfile {
	return UserProfile{
		UserID:    u.ID,
		Name:      u.Name,
		Email:     u.Email,
		Phone:     u.Phone,
		Role:      string(u.Role),
		IsActive:  u.IsActive,
		CreatedAt: u.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
