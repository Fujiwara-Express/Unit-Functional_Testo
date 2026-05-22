package user_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
	"pgregory.net/rapid"

	"user-service/internal/domain"
	"user-service/internal/service/user"
)

// ---------------------------------------------------------------------------
// In-memory repository fakes (shared with auth tests pattern)
// ---------------------------------------------------------------------------

type memUserRepo struct {
	mu      sync.Mutex
	users   map[string]domain.User
	byEmail map[string]string
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

// ---------------------------------------------------------------------------
// Generators
// ---------------------------------------------------------------------------

func genValidEmail(t *rapid.T) string {
	local := rapid.StringMatching(`[a-zA-Z0-9]{3,10}`).Draw(t, "local")
	dom := rapid.StringMatching(`[a-zA-Z0-9]{3,10}`).Draw(t, "domain")
	tld := rapid.StringMatching(`[a-zA-Z]{2,4}`).Draw(t, "tld")
	return local + "@" + dom + "." + tld
}

func genValidPassword(t *rapid.T) string {
	length := rapid.IntRange(8, 20).Draw(t, "pwdLen")
	runes := make([]rune, length)
	for i := range runes {
		runes[i] = rune(rapid.IntRange(33, 126).Draw(t, "char"))
	}
	return string(runes)
}

func genValidRole(t *rapid.T) domain.Role {
	return domain.Role(rapid.SampledFrom([]string{"CUSTOMER", "COURIER", "ADMIN"}).Draw(t, "role"))
}

func genValidName(t *rapid.T) string {
	return rapid.StringMatching(`[a-zA-Z ]{2,20}`).Draw(t, "name")
}

func genValidPhone(t *rapid.T) string {
	length := rapid.IntRange(7, 12).Draw(t, "phoneLen")
	digits := make([]byte, length)
	for i := range digits {
		digits[i] = byte('0' + rapid.IntRange(0, 9).Draw(t, "d"))
	}
	return string(digits)
}

// seedUser inserts a user directly into the repo and returns the stored user.
// The password hash corresponds to the plaintext "password1" at MinCost.
func seedUser(ctx context.Context, repo *memUserRepo, t *rapid.T) domain.User {
	hash, err := bcrypt.GenerateFromPassword([]byte("password1"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("seedUser: bcrypt failed: %v", err)
	}
	u := domain.User{
		Name:         genValidName(t),
		Email:        genValidEmail(t),
		Phone:        genValidPhone(t),
		PasswordHash: string(hash),
		Role:         genValidRole(t),
		IsActive:     true,
	}
	created, err := repo.Create(ctx, u)
	if err != nil {
		t.Fatalf("seedUser: Create failed: %v", err)
	}
	return created
}

// newService creates a fresh UserService backed by a new in-memory repo.
// Uses bcrypt.MinCost to keep property tests fast.
func newService() (user.UserService, *memUserRepo) {
	repo := newMemUserRepo()
	svc := user.NewWithCost(repo, bcrypt.MinCost)
	return svc, repo
}

// ---------------------------------------------------------------------------
// Property 11: Profile response never contains password hash
// Feature: user-service, Property 11: Profile response never contains password hash
// Validates: Requirements 4.1, 8.1
// ---------------------------------------------------------------------------

func TestProperty11_ProfileNeverContainsPasswordHash(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		// Feature: user-service, Property 11: Profile response never contains password hash
		// Validates: Requirements 4.1, 8.1
		svc, repo := newService()
		ctx := context.Background()

		u := seedUser(ctx, repo, rt)

		profile, err := svc.GetProfile(ctx, u.ID)
		if err != nil {
			rt.Fatalf("GetProfile failed: %v", err)
		}

		// Marshal to JSON and check no password_hash field is present
		data, err := json.Marshal(profile)
		if err != nil {
			rt.Fatalf("json.Marshal failed: %v", err)
		}

		var raw map[string]interface{}
		if err := json.Unmarshal(data, &raw); err != nil {
			rt.Fatalf("json.Unmarshal failed: %v", err)
		}

		if _, found := raw["password_hash"]; found {
			rt.Fatal("profile JSON must not contain password_hash field")
		}
	})
}

// ---------------------------------------------------------------------------
// Property 12: Valid profile update is reflected in the response
// Feature: user-service, Property 12: Valid profile update is reflected in the response
// Validates: Requirements 4.2, 4.3
// ---------------------------------------------------------------------------

func TestProperty12_ValidProfileUpdateReflectedInResponse(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		// Feature: user-service, Property 12: Valid profile update is reflected in the response
		// Validates: Requirements 4.2, 4.3
		svc, repo := newService()
		ctx := context.Background()

		u := seedUser(ctx, repo, rt)

		newName := genValidName(rt)
		newPhone := genValidPhone(rt)

		updateResp, err := svc.UpdateProfile(ctx, u.ID, user.UpdateProfileRequest{
			Name:  newName,
			Phone: newPhone,
		})
		if err != nil {
			rt.Fatalf("UpdateProfile failed: %v", err)
		}
		if updateResp.Status != "UPDATED" {
			rt.Fatalf("expected status UPDATED, got %q", updateResp.Status)
		}

		profile, err := svc.GetProfile(ctx, u.ID)
		if err != nil {
			rt.Fatalf("GetProfile after update failed: %v", err)
		}
		if profile.Name != newName {
			rt.Fatalf("expected name %q, got %q", newName, profile.Name)
		}
		if profile.Phone != newPhone {
			rt.Fatalf("expected phone %q, got %q", newPhone, profile.Phone)
		}
	})
}

// ---------------------------------------------------------------------------
// Property 14: Correct current password allows password change
// Feature: user-service, Property 14: Correct current password allows password change
// Validates: Requirements 5.1
// ---------------------------------------------------------------------------

// Remove the WithMaxRuns wrapper since we now use MinCost
func TestProperty14_CorrectCurrentPasswordAllowsChange(t *testing.T) {
	// Feature: user-service, Property 14: Correct current password allows password change
	// Validates: Requirements 5.1
	rapid.Check(t, func(rt *rapid.T) {
		svc, repo := newService()
		ctx := context.Background()

		// Use a known plaintext password so we can verify the change
		const knownPassword = "password1" // matches the seeded bcrypt hash
		u := seedUser(ctx, repo, rt)

		newPwd := genValidPassword(rt)
		// Ensure new password differs from current
		if newPwd == knownPassword {
			newPwd = newPwd + "X"
		}

		oldState, err := repo.FindByID(ctx, u.ID)
		if err != nil {
			rt.Fatalf("FindByID failed: %v", err)
		}

		err = svc.ChangePassword(ctx, u.ID, user.ChangePasswordRequest{
			CurrentPassword: knownPassword,
			NewPassword:     newPwd,
		})
		if err != nil {
			rt.Fatalf("ChangePassword failed unexpectedly: %v", err)
		}

		newState, err := repo.FindByID(ctx, u.ID)
		if err != nil {
			rt.Fatalf("FindByID after change failed: %v", err)
		}
		if newState.PasswordHash == oldState.PasswordHash {
			rt.Fatal("expected password hash to change after ChangePassword")
		}
	})
}

// ---------------------------------------------------------------------------
// Property 15: Wrong current password blocks password change
// Feature: user-service, Property 15: Wrong current password blocks password change
// Validates: Requirements 5.2
// ---------------------------------------------------------------------------

func TestProperty15_WrongCurrentPasswordBlocksChange(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		// Feature: user-service, Property 15: Wrong current password blocks password change
		// Validates: Requirements 5.2
		svc, repo := newService()
		ctx := context.Background()

		u := seedUser(ctx, repo, rt)

		// Generate a wrong password (guaranteed different from "password1")
		wrongPwd := genValidPassword(rt)
		if wrongPwd == "password1" {
			wrongPwd = wrongPwd + "X"
		}

		err := svc.ChangePassword(ctx, u.ID, user.ChangePasswordRequest{
			CurrentPassword: wrongPwd,
			NewPassword:     "newvalidpwd99",
		})
		if err == nil {
			rt.Fatal("expected auth error for wrong current password, got nil")
		}
		if !errors.Is(err, domain.ErrUnauthorized) {
			rt.Fatalf("expected ErrUnauthorized, got %v", err)
		}
	})
}

// ---------------------------------------------------------------------------
// Property 16: Account deactivation/reactivation round-trip
// Feature: user-service, Property 16: Account deactivation/reactivation round-trip
// Validates: Requirements 6.1, 6.2
// ---------------------------------------------------------------------------

func TestProperty16_DeactivationReactivationRoundTrip(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		// Feature: user-service, Property 16: Account deactivation/reactivation round-trip
		// Validates: Requirements 6.1, 6.2
		svc, repo := newService()
		ctx := context.Background()

		u := seedUser(ctx, repo, rt)

		// Deactivate
		if err := svc.SetAccountStatus(ctx, "ADMIN", u.ID, false); err != nil {
			rt.Fatalf("SetAccountStatus(false) failed: %v", err)
		}
		after, err := repo.FindByID(ctx, u.ID)
		if err != nil {
			rt.Fatalf("FindByID after deactivate failed: %v", err)
		}
		if after.IsActive {
			rt.Fatal("expected is_active=false after deactivation")
		}

		// Reactivate
		if err := svc.SetAccountStatus(ctx, "ADMIN", u.ID, true); err != nil {
			rt.Fatalf("SetAccountStatus(true) failed: %v", err)
		}
		after, err = repo.FindByID(ctx, u.ID)
		if err != nil {
			rt.Fatalf("FindByID after reactivate failed: %v", err)
		}
		if !after.IsActive {
			rt.Fatal("expected is_active=true after reactivation")
		}
	})
}

// ---------------------------------------------------------------------------
// Property 18: Non-admin cannot change account status
// Feature: user-service, Property 18: Non-admin cannot change account status
// Validates: Requirements 6.4
// ---------------------------------------------------------------------------

func TestProperty18_NonAdminCannotChangeAccountStatus(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		// Feature: user-service, Property 18: Non-admin cannot change account status
		// Validates: Requirements 6.4
		svc, repo := newService()
		ctx := context.Background()

		u := seedUser(ctx, repo, rt)

		nonAdminRole := rapid.SampledFrom([]string{"CUSTOMER", "COURIER"}).Draw(rt, "nonAdminRole")

		err := svc.SetAccountStatus(ctx, nonAdminRole, u.ID, false)
		if err == nil {
			rt.Fatal("expected forbidden error for non-admin, got nil")
		}
		if !errors.Is(err, domain.ErrForbidden) {
			rt.Fatalf("expected ErrForbidden, got %v", err)
		}
	})
}
