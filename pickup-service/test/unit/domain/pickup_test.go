package domain_test

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/pickup-service/internal/domain"
	"github.com/pickup-service/test/unit/fixtures"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func TestPickup_Transition_ValidTransitions(t *testing.T) {
	tests := []struct {
		name string
		from domain.Status
		to   domain.Status
	}{
		{"SCHEDULED -> ASSIGNED", domain.StatusScheduled, domain.StatusAssigned},
		{"SCHEDULED -> CANCELLED", domain.StatusScheduled, domain.StatusCancelled},
		{"ASSIGNED -> PICKED_UP", domain.StatusAssigned, domain.StatusPickedUp},
		{"ASSIGNED -> FAILED_ATTEMPT", domain.StatusAssigned, domain.StatusFailedAttempt},
		{"FAILED_ATTEMPT -> SCHEDULED", domain.StatusFailedAttempt, domain.StatusScheduled},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := fixtures.PickupWithStatus(tc.from)
			err := p.Transition(tc.to)
			require.NoError(t, err)
			assert.Equal(t, tc.to, p.Status)
		})
	}
}

func TestPickup_Transition_InvalidTransitions(t *testing.T) {
	tests := []struct {
		name string
		from domain.Status
		to   domain.Status
	}{
		{"SCHEDULED -> PICKED_UP (invalid)", domain.StatusScheduled, domain.StatusPickedUp},
		{"ASSIGNED -> SCHEDULED (invalid)", domain.StatusAssigned, domain.StatusScheduled},
		{"PICKED_UP -> ASSIGNED (invalid)", domain.StatusPickedUp, domain.StatusAssigned},
		{"CANCELLED -> SCHEDULED (invalid)", domain.StatusCancelled, domain.StatusScheduled},
		{"FAILED_ATTEMPT -> PICKED_UP (invalid)", domain.StatusFailedAttempt, domain.StatusPickedUp},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := fixtures.PickupWithStatus(tc.from)
			err := p.Transition(tc.to)
			require.Error(t, err)
			assert.Equal(t, tc.from, p.Status)
		})
	}
}

func TestPickup_Validate_MissingRequiredFields(t *testing.T) {
	tests := []struct {
		name          string
		mutate        func(p *domain.Pickup)
		wantErrSubstr string
	}{
		{
			name:          "Missing OrderID",
			mutate:        func(p *domain.Pickup) { p.OrderID = "" },
			wantErrSubstr: "order_id",
		},
		{
			name:          "Missing UserID",
			mutate:        func(p *domain.Pickup) { p.UserID = "" },
			wantErrSubstr: "user_id",
		},
		{
			name:          "Missing PickupAddress",
			mutate:        func(p *domain.Pickup) { p.PickupAddress = "" },
			wantErrSubstr: "pickup_address",
		},
		{
			name:          "Missing PickupCityCode",
			mutate:        func(p *domain.Pickup) { p.PickupCityCode = "" },
			wantErrSubstr: "pickup_city_code",
		},
		{
			name:          "Missing ContactName",
			mutate:        func(p *domain.Pickup) { p.ContactName = "" },
			wantErrSubstr: "contact_name",
		},
		{
			name:          "Missing ContactPhone",
			mutate:        func(p *domain.Pickup) { p.ContactPhone = "" },
			wantErrSubstr: "contact_phone",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := fixtures.ValidPickup()
			tc.mutate(p)
			err := p.Validate()
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErrSubstr)
		})
	}
}

func TestPickup_AttemptCount_IncrementOnFailedAttempt(t *testing.T) {
	p := fixtures.ValidPickup()
	require.Equal(t, 0, p.AttemptCount)

	for cycle := 1; cycle <= 3; cycle++ {
		// SCHEDULED -> ASSIGNED
		err := p.Transition(domain.StatusAssigned)
		require.NoError(t, err)
		assert.Equal(t, domain.StatusAssigned, p.Status)

		// ASSIGNED -> FAILED_ATTEMPT (AttemptCount should increment)
		err = p.Transition(domain.StatusFailedAttempt)
		require.NoError(t, err)
		assert.Equal(t, domain.StatusFailedAttempt, p.Status)
		assert.Equal(t, cycle, p.AttemptCount, "AttemptCount should be %d after cycle %d", cycle, cycle)

		// FAILED_ATTEMPT -> SCHEDULED (reset for next cycle, except last)
		if cycle < 3 {
			err = p.Transition(domain.StatusScheduled)
			require.NoError(t, err)
			assert.Equal(t, domain.StatusScheduled, p.Status)
		}
	}
}

// ── Property-Based Tests ──────────────────────────────────────────────────────

// generateValidPickup creates a random but fully-populated Pickup using rapid generators.
func generateValidPickup(t *rapid.T) *domain.Pickup {
	ts := func(name string) time.Time {
		return time.Unix(rapid.Int64Range(0, 1e9).Draw(t, name), 0).UTC()
	}
	return &domain.Pickup{
		PickupID:            rapid.StringMatching(`[a-z0-9]{4,16}`).Draw(t, "pickup_id"),
		OrderID:             rapid.StringMatching(`[a-z0-9]{4,16}`).Draw(t, "order_id"),
		UserID:              rapid.StringMatching(`[a-z0-9]{4,16}`).Draw(t, "user_id"),
		CourierID:           rapid.StringMatching(`[a-z0-9]{0,16}`).Draw(t, "courier_id"),
		Status:              domain.StatusScheduled,
		PickupAddress:       rapid.StringMatching(`[A-Za-z0-9 ]{4,32}`).Draw(t, "pickup_address"),
		PickupCityCode:      rapid.StringMatching(`[A-Z]{2,6}`).Draw(t, "pickup_city_code"),
		ContactName:         rapid.StringMatching(`[A-Za-z ]{4,32}`).Draw(t, "contact_name"),
		ContactPhone:        rapid.StringMatching(`\+[0-9]{8,14}`).Draw(t, "contact_phone"),
		AttemptCount:        rapid.IntRange(0, 10).Draw(t, "attempt_count"),
		EstimatedPickupTime: ts("estimated_pickup_time"),
		CreatedAt:           ts("created_at"),
		UpdatedAt:           ts("updated_at"),
	}
}

// Validates: Requirements 1.5
// Feature: pickup-service-unit-tests, Property 3: Pickup JSON round-trip
func TestPickup_JSON_RoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		original := generateValidPickup(t)

		data, err := json.Marshal(original)
		require.NoError(t, err)

		var restored domain.Pickup
		err = json.Unmarshal(data, &restored)
		require.NoError(t, err)

		assert.Equal(t, *original, restored)
	})
}

// validTransitionPairs lists all allowed (from, to) transitions.
var validTransitionPairs = [][2]domain.Status{
	{domain.StatusScheduled, domain.StatusAssigned},
	{domain.StatusScheduled, domain.StatusCancelled},
	{domain.StatusAssigned, domain.StatusPickedUp},
	{domain.StatusAssigned, domain.StatusFailedAttempt},
	{domain.StatusFailedAttempt, domain.StatusScheduled},
}

// allStatuses lists every known status.
var allStatuses = []domain.Status{
	domain.StatusScheduled,
	domain.StatusAssigned,
	domain.StatusPickedUp,
	domain.StatusFailedAttempt,
	domain.StatusCancelled,
}

// Validates: Requirements 1.2
// Feature: pickup-service-unit-tests, Property 1: Valid state transitions always succeed
func TestPickup_Transition_ValidAlwaysSucceed(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		idx := rapid.IntRange(0, len(validTransitionPairs)-1).Draw(t, "pair_idx")
		pair := validTransitionPairs[idx]
		from, to := pair[0], pair[1]

		p := fixtures.PickupWithStatus(from)
		err := p.Transition(to)
		require.NoError(t, err)
		assert.Equal(t, to, p.Status)
	})
}

// isValidTransition returns true when (from, to) is in the valid transition table.
func isValidTransition(from, to domain.Status) bool {
	for _, pair := range validTransitionPairs {
		if pair[0] == from && pair[1] == to {
			return true
		}
	}
	return false
}

// Validates: Requirements 1.3
// Feature: pickup-service-unit-tests, Property 2: Invalid state transitions always fail
func TestPickup_Transition_InvalidAlwaysFail(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		fromIdx := rapid.IntRange(0, len(allStatuses)-1).Draw(t, "from_idx")
		toIdx := rapid.IntRange(0, len(allStatuses)-1).Draw(t, "to_idx")
		from := allStatuses[fromIdx]
		to := allStatuses[toIdx]

		if isValidTransition(from, to) {
			t.Skip()
		}

		p := fixtures.PickupWithStatus(from)
		err := p.Transition(to)
		require.Error(t, err)
		assert.Equal(t, from, p.Status)
	})
}

// requiredFields maps field names to a mutator that clears that field.
var requiredFields = []struct {
	name  string
	clear func(p *domain.Pickup)
}{
	{"order_id", func(p *domain.Pickup) { p.OrderID = "" }},
	{"user_id", func(p *domain.Pickup) { p.UserID = "" }},
	{"pickup_address", func(p *domain.Pickup) { p.PickupAddress = "" }},
	{"pickup_city_code", func(p *domain.Pickup) { p.PickupCityCode = "" }},
	{"contact_name", func(p *domain.Pickup) { p.ContactName = "" }},
	{"contact_phone", func(p *domain.Pickup) { p.ContactPhone = "" }},
}

// Validates: Requirements 1.4
// Feature: pickup-service-unit-tests, Property 4: Missing required field produces named error
func TestPickup_Validate_MissingFieldNamedInError(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		idx := rapid.IntRange(0, len(requiredFields)-1).Draw(t, "field_idx")
		field := requiredFields[idx]

		p := fixtures.ValidPickup()
		field.clear(p)

		err := p.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), field.name)
	})
}

// Validates: Requirements 1.8
// Feature: pickup-service-unit-tests, Property 5: attempt_count equals number of FAILED_ATTEMPT transitions
func TestPickup_AttemptCount_EqualsFailedAttemptTransitions(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		n := rapid.IntRange(1, 5).Draw(t, "n_cycles")

		p := fixtures.ValidPickup() // starts SCHEDULED, AttemptCount == 0
		for i := 1; i <= n; i++ {
			err := p.Transition(domain.StatusAssigned)
			require.NoError(t, err)

			err = p.Transition(domain.StatusFailedAttempt)
			require.NoError(t, err)

			if i < n {
				err = p.Transition(domain.StatusScheduled)
				require.NoError(t, err)
			}
		}

		assert.Equal(t, n, p.AttemptCount)
	})
}
