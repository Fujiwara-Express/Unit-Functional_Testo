package http_test

import (
	"encoding/json"
	"testing"
	"time"

	"pgregory.net/rapid"

	"user-service/internal/domain"
)

// genUser produces a random domain.User with well-formed fields.
func genUser(t *rapid.T) domain.User {
	roles := []domain.Role{domain.RoleCustomer, domain.RoleCourier, domain.RoleAdmin}
	role := roles[rapid.IntRange(0, 2).Draw(t, "role_idx")]

	return domain.User{
		ID:           rapid.StringMatching(`[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}`).Draw(t, "id"),
		Name:         rapid.StringMatching(`[A-Za-z ]{1,40}`).Draw(t, "name"),
		Email:        rapid.StringMatching(`[a-z]{3,8}@[a-z]{3,6}\.[a-z]{2,4}`).Draw(t, "email"),
		Phone:        rapid.StringMatching(`\+?[0-9]{7,12}`).Draw(t, "phone"),
		PasswordHash: "$2a$10$hashedvalue",
		Role:         role,
		IsActive:     rapid.Bool().Draw(t, "is_active"),
		CreatedAt:    time.Now().UTC().Truncate(time.Second),
	}
}

// Feature: user-service, Property 20: JSON serialization round-trip
// Validates: Requirements 8.4
func TestProperty20_JSONSerializationRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		u := genUser(t)

		// First marshal
		data, err := json.Marshal(u)
		if err != nil {
			t.Fatalf("first marshal failed: %v", err)
		}

		// Unmarshal into a new User
		var u2 domain.User
		if err := json.Unmarshal(data, &u2); err != nil {
			t.Fatalf("unmarshal failed: %v", err)
		}

		// Re-marshal
		data2, err := json.Marshal(u2)
		if err != nil {
			t.Fatalf("second marshal failed: %v", err)
		}

		// The two JSON representations must be equivalent
		if string(data) != string(data2) {
			t.Fatalf("round-trip mismatch:\n  first:  %s\n  second: %s", data, data2)
		}

		// password_hash must not appear in the JSON output (json:"-" tag)
		var raw map[string]json.RawMessage
		if err := json.Unmarshal(data, &raw); err != nil {
			t.Fatalf("raw unmarshal failed: %v", err)
		}
		if _, found := raw["password_hash"]; found {
			t.Fatal("password_hash must not be present in JSON output")
		}
	})
}

// Feature: user-service, Property 21: Unknown JSON fields are ignored on deserialization
// Validates: Requirements 8.3
func TestProperty21_UnknownJSONFieldsIgnored(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		u := genUser(t)

		// Marshal the known user to a map so we can inject extra fields
		data, err := json.Marshal(u)
		if err != nil {
			t.Fatalf("marshal failed: %v", err)
		}

		var raw map[string]json.RawMessage
		if err := json.Unmarshal(data, &raw); err != nil {
			t.Fatalf("raw unmarshal failed: %v", err)
		}

		// Inject a random unknown field
		unknownKey := rapid.StringMatching(`[a-z]{5,10}_unknown`).Draw(t, "unknown_key")
		unknownVal, _ := json.Marshal(rapid.String().Draw(t, "unknown_val"))
		raw[unknownKey] = unknownVal

		// Re-marshal with the extra field
		augmented, err := json.Marshal(raw)
		if err != nil {
			t.Fatalf("augmented marshal failed: %v", err)
		}

		// Unmarshal into domain.User — unknown fields should be silently ignored
		var u2 domain.User
		if err := json.Unmarshal(augmented, &u2); err != nil {
			t.Fatalf("unmarshal with unknown field failed: %v", err)
		}

		// Known fields must still match
		if u2.ID != u.ID {
			t.Fatalf("user_id mismatch: got %q, want %q", u2.ID, u.ID)
		}
		if u2.Name != u.Name {
			t.Fatalf("name mismatch: got %q, want %q", u2.Name, u.Name)
		}
		if u2.Email != u.Email {
			t.Fatalf("email mismatch: got %q, want %q", u2.Email, u.Email)
		}
		if u2.Role != u.Role {
			t.Fatalf("role mismatch: got %q, want %q", u2.Role, u.Role)
		}
		if u2.IsActive != u.IsActive {
			t.Fatalf("is_active mismatch: got %v, want %v", u2.IsActive, u.IsActive)
		}
	})
}
