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
	"github.com/notification-service/internal/domain"
	"github.com/notification-service/internal/repository"
	"github.com/notification-service/test/unit/fixtures"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

// Validates: Requirements 2.1
func TestNotificationRepository_CreateNotificationLog_ValidInput(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	log := fixtures.ValidNotificationLog()
	log.NotifID = "notif-abc"

	rows := sqlmock.NewRows([]string{"notif_id"}).AddRow("notif-abc")
	mock.ExpectQuery(`INSERT INTO notification_logs`).WillReturnRows(rows)

	repo := repository.NewNotificationRepository(db)
	id, err := repo.CreateNotificationLog(context.Background(), log)
	require.NoError(t, err)
	assert.NotEmpty(t, id)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// Validates: Requirements 2.2
func TestNotificationRepository_GetNotificationLogByID_Exists(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	expected := fixtures.ValidNotificationLog()
	now := time.Now().UTC().Truncate(time.Second)
	expected.SentAt = now

	rows := sqlmock.NewRows([]string{
		"notif_id", "user_id", "tracking_number", "channel", "template_id", "message", "status", "sent_at",
	}).AddRow(
		expected.NotifID, expected.UserID, expected.TrackingNumber,
		string(expected.Channel), expected.TemplateID, expected.Message,
		string(expected.Status), expected.SentAt,
	)

	mock.ExpectQuery(`SELECT .* FROM notification_logs WHERE notif_id`).
		WithArgs(expected.NotifID).
		WillReturnRows(rows)

	repo := repository.NewNotificationRepository(db)
	got, err := repo.GetNotificationLogByID(context.Background(), expected.NotifID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, expected.NotifID, got.NotifID)
	assert.Equal(t, expected.UserID, got.UserID)
	assert.Equal(t, expected.Channel, got.Channel)
	assert.Equal(t, expected.Status, got.Status)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// Validates: Requirements 2.3
func TestNotificationRepository_GetNotificationLogByID_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery(`SELECT .* FROM notification_logs WHERE notif_id`).
		WithArgs("nonexistent-id").
		WillReturnError(sql.ErrNoRows)

	repo := repository.NewNotificationRepository(db)
	_, err = repo.GetNotificationLogByID(context.Background(), "nonexistent-id")
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrNotFound)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// Validates: Requirements 2.4
func TestNotificationRepository_UpdateNotificationLogStatus_ValidInput(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectExec(`UPDATE notification_logs`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	repo := repository.NewNotificationRepository(db)
	err = repo.UpdateNotificationLogStatus(context.Background(), "notif-123", domain.NotifStatusSent, time.Now())
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// Validates: Requirements 2.5
func TestNotificationRepository_ListTemplates_ReturnsAll(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	rows := sqlmock.NewRows([]string{"template_id", "event_type", "channel", "subject", "body_template"}).
		AddRow("tmpl-001", "OUT_FOR_DELIVERY", "PUSH", "Subject 1", "Body 1").
		AddRow("tmpl-002", "DELIVERED", "EMAIL", "Subject 2", "Body 2")

	mock.ExpectQuery(`SELECT .* FROM notification_templates`).WillReturnRows(rows)

	repo := repository.NewNotificationRepository(db)
	templates, err := repo.ListTemplates(context.Background())
	require.NoError(t, err)
	assert.Len(t, templates, 2)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// Validates: Requirements 2.6
func TestNotificationRepository_GetTemplateByID_Exists(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	expected := fixtures.ValidNotificationTemplate()

	rows := sqlmock.NewRows([]string{"template_id", "event_type", "channel", "subject", "body_template"}).
		AddRow(expected.TemplateID, expected.EventType, string(expected.Channel), expected.Subject, expected.BodyTemplate)

	mock.ExpectQuery(`SELECT .* FROM notification_templates WHERE template_id`).
		WithArgs(expected.TemplateID).
		WillReturnRows(rows)

	repo := repository.NewNotificationRepository(db)
	got, err := repo.GetTemplateByID(context.Background(), expected.TemplateID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, expected.TemplateID, got.TemplateID)
	assert.Equal(t, expected.Channel, got.Channel)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// Validates: Requirements 2.7
func TestNotificationRepository_GetTemplateByID_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery(`SELECT .* FROM notification_templates WHERE template_id`).
		WithArgs("nonexistent-tmpl").
		WillReturnError(sql.ErrNoRows)

	repo := repository.NewNotificationRepository(db)
	_, err = repo.GetTemplateByID(context.Background(), "nonexistent-tmpl")
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrNotFound)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// Validates: Requirements 2.8
func TestNotificationRepository_CreateTemplate_ValidInput(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	tmpl := fixtures.ValidNotificationTemplate()

	rows := sqlmock.NewRows([]string{"template_id"}).AddRow(tmpl.TemplateID)
	mock.ExpectQuery(`INSERT INTO notification_templates`).WillReturnRows(rows)

	repo := repository.NewNotificationRepository(db)
	id, err := repo.CreateTemplate(context.Background(), tmpl)
	require.NoError(t, err)
	assert.NotEmpty(t, id)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// Validates: Requirements 2.9
func TestNotificationRepository_UpdateTemplate_ValidInput(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectExec(`UPDATE notification_templates`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	repo := repository.NewNotificationRepository(db)
	err = repo.UpdateTemplate(context.Background(), "tmpl-001", "New Subject", "New body {{tracking_number}}")
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// Validates: Requirements 2.10
func TestNotificationRepository_ConnectionError_WrapsError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery(`SELECT .* FROM notification_logs WHERE notif_id`).
		WithArgs("notif-123").
		WillReturnError(sql.ErrConnDone)

	repo := repository.NewNotificationRepository(db)
	_, err = repo.GetNotificationLogByID(context.Background(), "notif-123")
	require.Error(t, err)
	assert.True(t,
		errors.Is(err, sql.ErrConnDone) || strings.Contains(err.Error(), sql.ErrConnDone.Error()),
		"expected error to wrap sql.ErrConnDone, got: %v", err,
	)
}

// Feature: notification-service-unit-tests, Property 6: Repository NotificationLog round-trip
// Validates: Requirements 2.1, 2.2
func TestNotificationRepository_NotificationLog_RoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		channels := []domain.Channel{domain.ChannelPush, domain.ChannelEmail, domain.ChannelWhatsApp}
		statuses := []domain.NotifStatus{domain.NotifStatusSent, domain.NotifStatusFailed, domain.NotifStatusPending}

		log := &domain.NotificationLog{
			NotifID:        rapid.StringMatching(`[a-z0-9-]{4,20}`).Draw(t, "notif_id"),
			UserID:         rapid.StringMatching(`[a-z0-9-]{4,20}`).Draw(t, "user_id"),
			TrackingNumber: rapid.StringMatching(`[A-Z0-9]{4,16}`).Draw(t, "tracking_number"),
			Channel:        channels[rapid.IntRange(0, 2).Draw(t, "channel_idx")],
			TemplateID:     rapid.StringMatching(`[a-z0-9-]{4,20}`).Draw(t, "template_id"),
			Message:        rapid.StringN(1, 100, -1).Draw(t, "message"),
			Status:         statuses[rapid.IntRange(0, 2).Draw(t, "status_idx")],
			SentAt:         time.Unix(rapid.Int64Range(0, 1e9).Draw(t, "sent_at"), 0).UTC(),
		}

		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		// Expect INSERT
		insertRows := sqlmock.NewRows([]string{"notif_id"}).AddRow(log.NotifID)
		mock.ExpectQuery(`INSERT INTO notification_logs`).WillReturnRows(insertRows)

		// Expect SELECT
		selectRows := sqlmock.NewRows([]string{
			"notif_id", "user_id", "tracking_number", "channel", "template_id", "message", "status", "sent_at",
		}).AddRow(
			log.NotifID, log.UserID, log.TrackingNumber,
			string(log.Channel), log.TemplateID, log.Message,
			string(log.Status), log.SentAt,
		)
		mock.ExpectQuery(`SELECT .* FROM notification_logs WHERE notif_id`).
			WithArgs(log.NotifID).
			WillReturnRows(selectRows)

		repo := repository.NewNotificationRepository(db)

		id, err := repo.CreateNotificationLog(context.Background(), log)
		require.NoError(t, err)
		assert.NotEmpty(t, id)

		got, err := repo.GetNotificationLogByID(context.Background(), log.NotifID)
		require.NoError(t, err)
		assert.Equal(t, log.NotifID, got.NotifID)
		assert.Equal(t, log.UserID, got.UserID)
		assert.Equal(t, log.Channel, got.Channel)
		assert.Equal(t, log.Status, got.Status)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// Feature: notification-service-unit-tests, Property 7: Repository NotificationTemplate round-trip
// Validates: Requirements 2.6, 2.8
func TestNotificationRepository_NotificationTemplate_RoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		channels := []domain.Channel{domain.ChannelPush, domain.ChannelEmail, domain.ChannelWhatsApp}

		tmpl := &domain.NotificationTemplate{
			TemplateID:   rapid.StringMatching(`[a-z0-9-]{4,20}`).Draw(t, "template_id"),
			EventType:    rapid.StringMatching(`[A-Z_]{4,20}`).Draw(t, "event_type"),
			Channel:      channels[rapid.IntRange(0, 2).Draw(t, "channel_idx")],
			Subject:      rapid.StringN(1, 100, -1).Draw(t, "subject"),
			BodyTemplate: rapid.StringN(1, 200, -1).Draw(t, "body_template"),
		}

		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		insertRows := sqlmock.NewRows([]string{"template_id"}).AddRow(tmpl.TemplateID)
		mock.ExpectQuery(`INSERT INTO notification_templates`).WillReturnRows(insertRows)

		selectRows := sqlmock.NewRows([]string{"template_id", "event_type", "channel", "subject", "body_template"}).
			AddRow(tmpl.TemplateID, tmpl.EventType, string(tmpl.Channel), tmpl.Subject, tmpl.BodyTemplate)
		mock.ExpectQuery(`SELECT .* FROM notification_templates WHERE template_id`).
			WithArgs(tmpl.TemplateID).
			WillReturnRows(selectRows)

		repo := repository.NewNotificationRepository(db)

		id, err := repo.CreateTemplate(context.Background(), tmpl)
		require.NoError(t, err)
		assert.NotEmpty(t, id)

		got, err := repo.GetTemplateByID(context.Background(), tmpl.TemplateID)
		require.NoError(t, err)
		assert.Equal(t, tmpl.TemplateID, got.TemplateID)
		assert.Equal(t, tmpl.Channel, got.Channel)
		assert.Equal(t, tmpl.Subject, got.Subject)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// Additional error-path tests to improve coverage

func TestNotificationRepository_UpdateNotificationLogStatus_Error(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectExec(`UPDATE notification_logs`).WillReturnError(sql.ErrConnDone)

	repo := repository.NewNotificationRepository(db)
	err = repo.UpdateNotificationLogStatus(context.Background(), "notif-123", domain.NotifStatusSent, time.Now())
	require.Error(t, err)
}

func TestNotificationRepository_ListTemplates_ScanError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery(`SELECT .* FROM notification_templates`).WillReturnError(sql.ErrConnDone)

	repo := repository.NewNotificationRepository(db)
	_, err = repo.ListTemplates(context.Background())
	require.Error(t, err)
}

func TestNotificationRepository_CreateTemplate_Error(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery(`INSERT INTO notification_templates`).WillReturnError(sql.ErrConnDone)

	repo := repository.NewNotificationRepository(db)
	_, err = repo.CreateTemplate(context.Background(), fixtures.ValidNotificationTemplate())
	require.Error(t, err)
}

func TestNotificationRepository_UpdateTemplate_Error(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectExec(`UPDATE notification_templates`).WillReturnError(sql.ErrConnDone)

	repo := repository.NewNotificationRepository(db)
	err = repo.UpdateTemplate(context.Background(), "tmpl-001", "Subject", "Body")
	require.Error(t, err)
}

func TestNotificationRepository_CreateNotificationLog_Error(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery(`INSERT INTO notification_logs`).WillReturnError(sql.ErrConnDone)

	repo := repository.NewNotificationRepository(db)
	_, err = repo.CreateNotificationLog(context.Background(), fixtures.ValidNotificationLog())
	require.Error(t, err)
}
