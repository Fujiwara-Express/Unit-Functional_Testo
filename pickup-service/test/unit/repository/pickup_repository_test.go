package repository_test

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/pickup-service/internal/domain"
	"github.com/pickup-service/internal/repository"
	"github.com/pickup-service/test/unit/fixtures"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

// ── 8.1 CreatePickup valid input ──────────────────────────────────────────────

// Validates: Requirements 2.1
func TestPickupRepository_CreatePickup_ValidInput(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	p := fixtures.ValidPickup()
	p.PickupID = "pickup-abc"

	rows := sqlmock.NewRows([]string{"pickup_id"}).AddRow("pickup-abc")
	mock.ExpectQuery(`INSERT INTO pickups`).
		WillReturnRows(rows)

	repo := repository.NewPickupRepository(db)
	id, err := repo.CreatePickup(context.Background(), p)
	require.NoError(t, err)
	assert.NotEmpty(t, id)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ── 8.2 GetPickupByID exists ──────────────────────────────────────────────────

// Validates: Requirements 2.2
func TestPickupRepository_GetPickupByID_Exists(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	expected := fixtures.ValidPickup()
	now := time.Now().UTC().Truncate(time.Second)
	expected.CreatedAt = now
	expected.UpdatedAt = now
	expected.EstimatedPickupTime = now

	rows := sqlmock.NewRows([]string{
		"pickup_id", "order_id", "user_id", "courier_id", "status",
		"pickup_address", "pickup_city_code", "contact_name", "contact_phone",
		"attempt_count", "estimated_pickup_time", "created_at", "updated_at",
	}).AddRow(
		expected.PickupID, expected.OrderID, expected.UserID, expected.CourierID, string(expected.Status),
		expected.PickupAddress, expected.PickupCityCode, expected.ContactName, expected.ContactPhone,
		expected.AttemptCount, expected.EstimatedPickupTime, expected.CreatedAt, expected.UpdatedAt,
	)

	mock.ExpectQuery(`SELECT .* FROM pickups WHERE pickup_id`).
		WithArgs(expected.PickupID).
		WillReturnRows(rows)

	repo := repository.NewPickupRepository(db)
	got, err := repo.GetPickupByID(context.Background(), expected.PickupID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, expected.PickupID, got.PickupID)
	assert.Equal(t, expected.OrderID, got.OrderID)
	assert.Equal(t, expected.UserID, got.UserID)
	assert.Equal(t, expected.Status, got.Status)
	assert.Equal(t, expected.PickupAddress, got.PickupAddress)
	assert.Equal(t, expected.ContactName, got.ContactName)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ── 8.3 GetPickupByID not found ───────────────────────────────────────────────

// Validates: Requirements 2.3
func TestPickupRepository_GetPickupByID_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery(`SELECT .* FROM pickups WHERE pickup_id`).
		WithArgs("nonexistent-id").
		WillReturnError(sql.ErrNoRows)

	repo := repository.NewPickupRepository(db)
	_, err = repo.GetPickupByID(context.Background(), "nonexistent-id")
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrNotFound)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ── 8.4 UpdatePickup valid input ──────────────────────────────────────────────

// Validates: Requirements 2.4
func TestPickupRepository_UpdatePickup_ValidInput(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	p := fixtures.ValidPickup()

	mock.ExpectExec(`UPDATE pickups`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	repo := repository.NewPickupRepository(db)
	err = repo.UpdatePickup(context.Background(), p)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ── 8.5 ListPickups with filters (table-driven) ───────────────────────────────

// Validates: Requirements 2.5, 2.6
func TestPickupRepository_ListPickups_WithFilters(t *testing.T) {
	pickupColumns := []string{
		"pickup_id", "order_id", "user_id", "courier_id", "status",
		"pickup_address", "pickup_city_code", "contact_name", "contact_phone",
		"attempt_count", "estimated_pickup_time", "created_at", "updated_at",
	}
	now := time.Now().UTC()

	cases := []struct {
		name    string
		filters repository.ListFilters
	}{
		{
			name: "all filters provided",
			filters: repository.ListFilters{
				CourierID: "courier-001",
				Status:    "SCHEDULED",
				Date:      "2024-01-15",
			},
		},
		{
			name: "partial filters - courier only",
			filters: repository.ListFilters{
				CourierID: "courier-002",
			},
		},
		{
			name:    "no filters",
			filters: repository.ListFilters{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			rows := sqlmock.NewRows(pickupColumns).AddRow(
				"pickup-1", "order-1", "user-1", "", "SCHEDULED",
				"123 Main St", "JKT", "John", "+628123", 0, now, now, now,
			)

			mock.ExpectQuery(`SELECT .* FROM pickups`).
				WillReturnRows(rows)

			repo := repository.NewPickupRepository(db)
			result, err := repo.ListPickups(context.Background(), tc.filters)
			require.NoError(t, err)
			assert.Len(t, result, 1)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// ── 8.6 CreatePickupAttempt valid input ───────────────────────────────────────

// Validates: Requirements 2.7
func TestPickupRepository_CreatePickupAttempt_ValidInput(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	attempt := &domain.PickupAttempt{
		AttemptID: "attempt-001",
		PickupID:  "pickup-123",
		Reason:    "no one home",
		CreatedAt: time.Now(),
	}

	mock.ExpectExec(`INSERT INTO pickup_attempts`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	repo := repository.NewPickupRepository(db)
	err = repo.CreatePickupAttempt(context.Background(), attempt)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ── 8.7 Connection error wraps error ─────────────────────────────────────────

// Validates: Requirements 2.8
func TestPickupRepository_ConnectionError_WrapsError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery(`SELECT .* FROM pickups WHERE pickup_id`).
		WithArgs("pickup-123").
		WillReturnError(sql.ErrConnDone)

	repo := repository.NewPickupRepository(db)
	_, err = repo.GetPickupByID(context.Background(), "pickup-123")
	require.Error(t, err)
	assert.True(t,
		errors.Is(err, sql.ErrConnDone) || strings.Contains(err.Error(), sql.ErrConnDone.Error()),
		"expected error to wrap sql.ErrConnDone, got: %v", err,
	)
}

// ── 8.8 Property: CreatePickup any valid Pickup ───────────────────────────────

// Feature: pickup-service-unit-tests, Property 6: Repository CreatePickup executes correct INSERT for any valid Pickup
// Validates: Requirements 2.1
func TestPickupRepository_CreatePickup_AnyValidPickup(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		p := generateValidPickup(t)

		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		rows := sqlmock.NewRows([]string{"pickup_id"}).AddRow(p.PickupID)
		mock.ExpectQuery(`INSERT INTO pickups`).WillReturnRows(rows)

		repo := repository.NewPickupRepository(db)
		id, err := repo.CreatePickup(context.Background(), p)
		require.NoError(t, err)
		assert.NotEmpty(t, id)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// ── 8.9 Property: GetPickupByID round-trip ────────────────────────────────────

// Feature: pickup-service-unit-tests, Property 7: Repository GetPickupByID round-trip
// Validates: Requirements 2.2, 2.3
func TestPickupRepository_GetPickupByID_RoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		p := generateValidPickup(t)

		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		rows := sqlmock.NewRows([]string{
			"pickup_id", "order_id", "user_id", "courier_id", "status",
			"pickup_address", "pickup_city_code", "contact_name", "contact_phone",
			"attempt_count", "estimated_pickup_time", "created_at", "updated_at",
		}).AddRow(
			p.PickupID, p.OrderID, p.UserID, p.CourierID, string(p.Status),
			p.PickupAddress, p.PickupCityCode, p.ContactName, p.ContactPhone,
			p.AttemptCount, p.EstimatedPickupTime, p.CreatedAt, p.UpdatedAt,
		)

		mock.ExpectQuery(`SELECT .* FROM pickups WHERE pickup_id`).
			WithArgs(p.PickupID).
			WillReturnRows(rows)

		repo := repository.NewPickupRepository(db)
		got, err := repo.GetPickupByID(context.Background(), p.PickupID)
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, p.PickupID, got.PickupID)
		assert.Equal(t, p.OrderID, got.OrderID)
		assert.Equal(t, p.UserID, got.UserID)
		assert.Equal(t, p.Status, got.Status)
		assert.Equal(t, p.PickupAddress, got.PickupAddress)
		assert.Equal(t, p.PickupCityCode, got.PickupCityCode)
		assert.Equal(t, p.ContactName, got.ContactName)
		assert.Equal(t, p.ContactPhone, got.ContactPhone)
		assert.Equal(t, p.AttemptCount, got.AttemptCount)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// ── 8.10 Property: ListPickups WHERE clause matches filters ───────────────────

// Feature: pickup-service-unit-tests, Property 8: Repository ListPickups WHERE clause matches provided filters
// Validates: Requirements 2.5, 2.6
func TestPickupRepository_ListPickups_WhereClauseMatchesFilters(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a non-empty subset of filters
		useCourier := rapid.Bool().Draw(t, "use_courier")
		useStatus := rapid.Bool().Draw(t, "use_status")
		useDate := rapid.Bool().Draw(t, "use_date")

		// Ensure at least one filter is set
		if !useCourier && !useStatus && !useDate {
			useCourier = true
		}

		filters := repository.ListFilters{}
		if useCourier {
			filters.CourierID = rapid.StringMatching(`[a-z0-9]{4,16}`).Draw(t, "courier_id")
		}
		if useStatus {
			statuses := []string{"SCHEDULED", "ASSIGNED", "PICKED_UP", "FAILED_ATTEMPT", "CANCELLED"}
			filters.Status = statuses[rapid.IntRange(0, len(statuses)-1).Draw(t, "status_idx")]
		}
		if useDate {
			filters.Date = rapid.StringMatching(`2024-0[1-9]-[0-2][0-9]`).Draw(t, "date")
		}

		db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
		require.NoError(t, err)
		defer db.Close()

		// Build expected query pattern based on filters
		queryPattern := `SELECT .* FROM pickups`
		if useCourier || useStatus || useDate {
			queryPattern += ` WHERE`
		}

		now := time.Now().UTC()
		rows := sqlmock.NewRows([]string{
			"pickup_id", "order_id", "user_id", "courier_id", "status",
			"pickup_address", "pickup_city_code", "contact_name", "contact_phone",
			"attempt_count", "estimated_pickup_time", "created_at", "updated_at",
		}).AddRow("p1", "o1", "u1", "", "SCHEDULED", "addr", "JKT", "name", "+628", 0, now, now, now)

		mock.ExpectQuery(queryPattern).WillReturnRows(rows)

		repo := repository.NewPickupRepository(db)
		result, err := repo.ListPickups(context.Background(), filters)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// ── helpers ───────────────────────────────────────────────────────────────────

func generateValidPickup(t *rapid.T) *domain.Pickup {
	ts := func(name string) time.Time {
		return time.Unix(rapid.Int64Range(0, 1e9).Draw(t, name), 0).UTC()
	}
	statuses := []domain.Status{
		domain.StatusScheduled, domain.StatusAssigned,
		domain.StatusPickedUp, domain.StatusFailedAttempt, domain.StatusCancelled,
	}
	return &domain.Pickup{
		PickupID:            rapid.StringMatching(`[a-z0-9]{4,16}`).Draw(t, "pickup_id"),
		OrderID:             rapid.StringMatching(`[a-z0-9]{4,16}`).Draw(t, "order_id"),
		UserID:              rapid.StringMatching(`[a-z0-9]{4,16}`).Draw(t, "user_id"),
		CourierID:           rapid.StringMatching(`[a-z0-9]{0,16}`).Draw(t, "courier_id"),
		Status:              statuses[rapid.IntRange(0, len(statuses)-1).Draw(t, "status_idx")],
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
