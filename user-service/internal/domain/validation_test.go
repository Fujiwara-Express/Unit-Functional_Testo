package domain_test

import (
	"strings"
	"testing"
	"unicode"

	"pgregory.net/rapid"

	"user-service/internal/domain"
)

// ---------------------------------------------------------------------------
// Generators
// ---------------------------------------------------------------------------

// genNonEmailString produces strings that are guaranteed NOT to be valid emails.
// Strategy: generate arbitrary strings and filter out anything that looks like
// a valid email (contains exactly one '@' with non-empty local and domain parts
// that include a dot in the domain).
func genNonEmailString(t *rapid.T) string {
	// Build strings that structurally cannot be valid emails.
	strategy := rapid.IntRange(0, 3).Draw(t, "strategy")
	switch strategy {
	case 0:
		// Plain word with no '@'
		return rapid.StringMatching(`[a-zA-Z0-9]{1,20}`).Draw(t, "word")
	case 1:
		// Multiple '@' signs
		local := rapid.StringMatching(`[a-zA-Z0-9]{1,10}`).Draw(t, "local")
		return local + "@@example.com"
	case 2:
		// '@' present but no TLD dot in domain
		local := rapid.StringMatching(`[a-zA-Z0-9]{1,10}`).Draw(t, "local")
		return local + "@nodot"
	default:
		// Empty string
		return ""
	}
}

// genShortPassword produces passwords of length 0–7.
func genShortPassword(t *rapid.T) string {
	length := rapid.IntRange(0, 7).Draw(t, "length")
	runes := make([]rune, length)
	for i := range runes {
		runes[i] = rune(rapid.IntRange(33, 126).Draw(t, "char")) // printable ASCII
	}
	return string(runes)
}

// genNonRoleString produces strings that are not valid Role constants.
func genNonRoleString(t *rapid.T) string {
	valid := map[string]bool{"CUSTOMER": true, "COURIER": true, "ADMIN": true}
	s := rapid.StringOf(rapid.RuneFrom(nil, unicode.Letter, unicode.Digit)).Draw(t, "roleStr")
	if valid[strings.ToUpper(s)] {
		// Append a character that makes it invalid
		s += "_INVALID"
	}
	return s
}

// genNonPhoneString produces strings that are guaranteed NOT to match the phone regex.
func genNonPhoneString(t *rapid.T) string {
	strategy := rapid.IntRange(0, 3).Draw(t, "strategy")
	switch strategy {
	case 0:
		// Contains letters
		return rapid.StringMatching(`[a-zA-Z]{3,10}`).Draw(t, "letters")
	case 1:
		// Too short (< 7 digits)
		length := rapid.IntRange(1, 6).Draw(t, "len")
		digits := make([]byte, length)
		for i := range digits {
			digits[i] = byte('0' + rapid.IntRange(0, 9).Draw(t, "d"))
		}
		return string(digits)
	case 2:
		// Too long (> 15 digits)
		length := rapid.IntRange(16, 25).Draw(t, "len")
		digits := make([]byte, length)
		for i := range digits {
			digits[i] = byte('0' + rapid.IntRange(0, 9).Draw(t, "d"))
		}
		return string(digits)
	default:
		// Special characters that aren't '+' or digits
		return "abc-def"
	}
}

// ---------------------------------------------------------------------------
// Property 2: Invalid email is always rejected
// Feature: user-service, Property 2: Invalid email is always rejected
// Validates: Requirements 1.3
// ---------------------------------------------------------------------------

func TestProperty2_InvalidEmailAlwaysRejected(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Feature: user-service, Property 2: Invalid email is always rejected
		// Validates: Requirements 1.3
		email := genNonEmailString(t)
		err := domain.ValidateEmail(email)
		if err == nil {
			t.Fatalf("expected ValidateEmail(%q) to return an error, got nil", email)
		}
	})
}

// ---------------------------------------------------------------------------
// Property 3: Short password is always rejected
// Feature: user-service, Property 3: Short password is always rejected
// Validates: Requirements 1.4, 5.3
// ---------------------------------------------------------------------------

func TestProperty3_ShortPasswordAlwaysRejected(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Feature: user-service, Property 3: Short password is always rejected
		// Validates: Requirements 1.4, 5.3
		pwd := genShortPassword(t)
		err := domain.ValidatePassword(pwd)
		if err == nil {
			t.Fatalf("expected ValidatePassword(%q) (len=%d) to return an error, got nil", pwd, len(pwd))
		}
	})
}

// ---------------------------------------------------------------------------
// Property 4: Invalid role is always rejected
// Feature: user-service, Property 4: Invalid role is always rejected
// Validates: Requirements 1.5
// ---------------------------------------------------------------------------

func TestProperty4_InvalidRoleAlwaysRejected(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Feature: user-service, Property 4: Invalid role is always rejected
		// Validates: Requirements 1.5
		roleStr := genNonRoleString(t)
		err := domain.ValidateRole(domain.Role(roleStr))
		if err == nil {
			t.Fatalf("expected ValidateRole(%q) to return an error, got nil", roleStr)
		}
	})
}

// ---------------------------------------------------------------------------
// Property 13: Invalid phone number is always rejected
// Feature: user-service, Property 13: Invalid phone number is always rejected
// Validates: Requirements 4.4
// ---------------------------------------------------------------------------

func TestProperty13_InvalidPhoneAlwaysRejected(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Feature: user-service, Property 13: Invalid phone number is always rejected
		// Validates: Requirements 4.4
		phone := genNonPhoneString(t)
		err := domain.ValidatePhone(phone)
		if err == nil {
			t.Fatalf("expected ValidatePhone(%q) to return an error, got nil", phone)
		}
	})
}
