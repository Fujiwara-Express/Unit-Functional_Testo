package auth_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
	"pgregory.net/rapid"

	"user-service/internal/domain"
	"user-service/internal/service/auth"
	"user-service/internal/token"
)

// ---------------------------------------------------------------------------
// In-memory repository fakes (real logic, no external dependencies)
// ---------------------------------------------------------------------------

type memUserRepo struct {
	mu      sync.Mutex
	users   map[string]domain.User // keyed by ID
	byEmail map[string]string      // email -> ID
	seq     int
}

func newMemUserRepo() *memUserRepo {
	return &memUserRepo{
		users:   make(map[string]domain.User),
		byEmail: make(map[string]string),
	}
}

func (r *memUserRepo) Create(ctx context.Context, u domain.User) (domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.byEmail[u.Email]; exists {
		return domain.User{}, domain.ErrEmailConflict
	}
	r.seq++
	u.ID = fmt.Sprintf("USR%04d", r.seq)
	u.CreatedAt = time.Now()
	r.users[u.ID] = u
	r.byEmail[u.Email] = u.ID
	return u, nil
}

func (r *memUserRepo) FindByEmail(ctx context.Context, email string) (domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	id, ok := r.byEmail[email]
	if !ok {
		return domain.User{}, domain.ErrNotFound
	}
	return r.users[id], nil
}

func (r *memUserRepo) FindByID(ctx context.Context, id string) (domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	u, ok := r.users[id]
	if !ok {
		return domain.User{}, domain.ErrNotFound
	}
	return u, nil
}

func (r *memUserRepo) Update(ctx context.Context, u domain.User) (domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.users[u.ID]; !ok {
		return domain.User{}, domain.ErrNotFound
	}
	r.users[u.ID] = u
	return u, nil
}

func (r *memUserRepo) SetStatus(ctx context.Context, id string, active bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	u, ok := r.users[id]
	if !ok {
		return domain.ErrNotFound
	}
	u.IsActive = active
	r.users[id] = u
	return nil
}

type memTokenRepo struct {
	mu     sync.Mutex
	tokens map[string]domain.RefreshToken // keyed by hash
}

func newMemTokenRepo() *memTokenRepo {
	return &memTokenRepo{tokens: make(map[string]domain.RefreshToken)}
}

func (r *memTokenRepo) Save(ctx context.Context, t domain.RefreshToken) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tokens[t.TokenHash] = t
	return nil
}

func (r *memTokenRepo) FindByTokenHash(ctx context.Context, hash string) (domain.RefreshToken, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.tokens[hash]
	if !ok {
		return domain.RefreshToken{}, domain.ErrNotFound
	}
	return t, nil
}

func (r *memTokenRepo) Revoke(ctx context.Context, hash string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.tokens[hash]; !ok {
		return domain.ErrNotFound
	}
	delete(r.tokens, hash)
	return nil
}

func (r *memTokenRepo) DeleteExpired(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now()
	for k, v := range r.tokens {
		if v.ExpiresAt.Before(now) {
			delete(r.tokens, k)
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

const testSecret = "test-secret-for-auth-service-props"

func newService() (auth.AuthService, *memUserRepo, *memTokenRepo) {
	ur := newMemUserRepo()
	tr := newMemTokenRepo()
	mgr := token.NewJWTManager(testSecret)
	svc := auth.NewWithCost(ur, tr, mgr, bcrypt.MinCost)
	return svc, ur, tr
}

func hashToken(t string) string {
	sum := sha256.Sum256([]byte(t))
	return hex.EncodeToString(sum[:])
}

// genValidEmail produces a structurally valid email string.
func genValidEmail(t *rapid.T) string {
	local := rapid.StringMatching(`[a-zA-Z0-9]{3,10}`).Draw(t, "local")
	domain_ := rapid.StringMatching(`[a-zA-Z0-9]{3,10}`).Draw(t, "domain")
	tld := rapid.StringMatching(`[a-zA-Z]{2,4}`).Draw(t, "tld")
	return local + "@" + domain_ + "." + tld
}

// genValidPassword produces a password of length >= 8.
func genValidPassword(t *rapid.T) string {
	length := rapid.IntRange(8, 20).Draw(t, "pwdLen")
	runes := make([]rune, length)
	for i := range runes {
		runes[i] = rune(rapid.IntRange(33, 126).Draw(t, "char"))
	}
	return string(runes)
}

// genValidRole picks one of the three valid roles.
func genValidRole(t *rapid.T) string {
	return rapid.SampledFrom([]string{"CUSTOMER", "COURIER", "ADMIN"}).Draw(t, "role")
}

// genValidName produces a non-empty name string.
func genValidName(t *rapid.T) string {
	return rapid.StringMatching(`[a-zA-Z ]{2,20}`).Draw(t, "name")
}

// genValidPhone produces a valid E.164-style phone number.
func genValidPhone(t *rapid.T) string {
	length := rapid.IntRange(7, 12).Draw(t, "phoneLen")
	digits := make([]byte, length)
	for i := range digits {
		digits[i] = byte('0' + rapid.IntRange(0, 9).Draw(t, "d"))
	}
	return string(digits)
}

// registerUser is a helper that registers a user and returns the response.
func registerUser(ctx context.Context, svc auth.AuthService, t *rapid.T) (auth.RegisterResponse, auth.RegisterRequest) {
	req := auth.RegisterRequest{
		Name:     genValidName(t),
		Email:    genValidEmail(t),
		Phone:    genValidPhone(t),
		Password: genValidPassword(t),
		Role:     genValidRole(t),
	}
	resp, err := svc.Register(ctx, req)
	if err != nil {
		t.Fatalf("Register failed unexpectedly: %v", err)
	}
	return resp, req
}

// ---------------------------------------------------------------------------
// Property 1: Valid registration always creates a user
// Feature: user-service, Property 1: Valid registration always creates a user
// Validates: Requirements 1.1
// ---------------------------------------------------------------------------

func TestProperty1_ValidRegistrationAlwaysCreatesUser(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		// Feature: user-service, Property 1: Valid registration always creates a user
		// Validates: Requirements 1.1
		svc, _, _ := newService()
		ctx := context.Background()

		resp, _ := registerUser(ctx, svc, rt)

		if resp.UserID == "" {
			rt.Fatal("expected non-empty user_id after registration")
		}
		if resp.Status != "CREATED" {
			rt.Fatalf("expected status CREATED, got %q", resp.Status)
		}
	})
}

// ---------------------------------------------------------------------------
// Property 5: Password is never stored in plaintext
// Feature: user-service, Property 5: Password is never stored in plaintext
// Validates: Requirements 1.6
// ---------------------------------------------------------------------------

func TestProperty5_PasswordNeverStoredInPlaintext(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		// Feature: user-service, Property 5: Password is never stored in plaintext
		// Validates: Requirements 1.6
		svc, ur, _ := newService()
		ctx := context.Background()

		resp, req := registerUser(ctx, svc, rt)

		stored, err := ur.FindByID(ctx, resp.UserID)
		if err != nil {
			rt.Fatalf("FindByID failed: %v", err)
		}
		if stored.PasswordHash == req.Password {
			rt.Fatalf("password stored in plaintext for user %q", resp.UserID)
		}
		if stored.PasswordHash == "" {
			rt.Fatal("password_hash is empty")
		}
	})
}

// ---------------------------------------------------------------------------
// Property 6: Valid login always returns a token pair
// Feature: user-service, Property 6: Valid login always returns a token pair
// Validates: Requirements 2.1
// ---------------------------------------------------------------------------

func TestProperty6_ValidLoginAlwaysReturnsTokenPair(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		// Feature: user-service, Property 6: Valid login always returns a token pair
		// Validates: Requirements 2.1
		svc, _, _ := newService()
		ctx := context.Background()

		_, req := registerUser(ctx, svc, rt)

		loginResp, err := svc.Login(ctx, auth.LoginRequest{
			Email:    req.Email,
			Password: req.Password,
		})
		if err != nil {
			rt.Fatalf("Login failed unexpectedly: %v", err)
		}
		if loginResp.AccessToken == "" {
			rt.Fatal("expected non-empty access_token")
		}
		if loginResp.RefreshToken == "" {
			rt.Fatal("expected non-empty refresh_token")
		}
		if loginResp.ExpiresIn <= 0 {
			rt.Fatalf("expected positive expires_in, got %d", loginResp.ExpiresIn)
		}
	})
}

// ---------------------------------------------------------------------------
// Property 7: Wrong password always fails login
// Feature: user-service, Property 7: Wrong password always fails login
// Validates: Requirements 2.3
// ---------------------------------------------------------------------------

func TestProperty7_WrongPasswordAlwaysFailsLogin(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		// Feature: user-service, Property 7: Wrong password always fails login
		// Validates: Requirements 2.3
		svc, _, _ := newService()
		ctx := context.Background()

		_, req := registerUser(ctx, svc, rt)

		// Generate a different password
		wrongPwd := genValidPassword(rt)
		// Ensure it's actually different
		if wrongPwd == req.Password {
			wrongPwd = wrongPwd + "X"
		}

		_, err := svc.Login(ctx, auth.LoginRequest{
			Email:    req.Email,
			Password: wrongPwd,
		})
		if err == nil {
			rt.Fatal("expected auth error for wrong password, got nil")
		}
		if !errors.Is(err, domain.ErrUnauthorized) {
			rt.Fatalf("expected ErrUnauthorized, got %v", err)
		}
	})
}

// ---------------------------------------------------------------------------
// Property 17: Deactivated user cannot authenticate
// Feature: user-service, Property 17: Deactivated user cannot authenticate
// Validates: Requirements 6.3
// ---------------------------------------------------------------------------

func TestProperty17_DeactivatedUserCannotAuthenticate(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		// Feature: user-service, Property 17: Deactivated user cannot authenticate
		// Validates: Requirements 6.3
		svc, ur, _ := newService()
		ctx := context.Background()

		resp, req := registerUser(ctx, svc, rt)

		// Deactivate the user directly via the repo
		if err := ur.SetStatus(ctx, resp.UserID, false); err != nil {
			rt.Fatalf("SetStatus failed: %v", err)
		}

		_, err := svc.Login(ctx, auth.LoginRequest{
			Email:    req.Email,
			Password: req.Password,
		})
		if err == nil {
			rt.Fatal("expected error for deactivated user, got nil")
		}
		if !errors.Is(err, domain.ErrForbidden) {
			rt.Fatalf("expected ErrForbidden, got %v", err)
		}
	})
}

// ---------------------------------------------------------------------------
// Property 19: Logout revokes the refresh token
// Feature: user-service, Property 19: Logout revokes the refresh token
// Validates: Requirements (Logout)
// ---------------------------------------------------------------------------

func TestProperty19_LogoutRevokesRefreshToken(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		// Feature: user-service, Property 19: Logout revokes the refresh token
		// Validates: Requirements (Logout)
		svc, _, _ := newService()
		ctx := context.Background()

		_, req := registerUser(ctx, svc, rt)

		loginResp, err := svc.Login(ctx, auth.LoginRequest{
			Email:    req.Email,
			Password: req.Password,
		})
		if err != nil {
			rt.Fatalf("Login failed: %v", err)
		}

		// Logout should succeed
		if err := svc.Logout(ctx, loginResp.RefreshToken); err != nil {
			rt.Fatalf("Logout failed: %v", err)
		}

		// Subsequent refresh with the same token must fail
		_, err = svc.RefreshToken(ctx, loginResp.RefreshToken)
		if err == nil {
			rt.Fatal("expected error after logout, got nil")
		}
		if !errors.Is(err, domain.ErrUnauthorized) {
			rt.Fatalf("expected ErrUnauthorized after logout, got %v", err)
		}
	})
}
