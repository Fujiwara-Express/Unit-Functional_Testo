package domain_test

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/notification-service/internal/domain"
	"github.com/notification-service/test/unit/fixtures"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

// Validates: Requirements 1.1
func TestNotificationLog_Fields_AllPresent(t *testing.T) {
	log := fixtures.ValidNotificationLog()
	assert.NotEmpty(t, log.NotifID)
	assert.NotEmpty(t, log.UserID)
	assert.NotEmpty(t, log.TrackingNumber)
	assert.NotEmpty(t, string(log.Channel))
	assert.NotEmpty(t, log.TemplateID)
	assert.NotEmpty(t, log.Message)
	assert.NotEmpty(t, string(log.Status))
	assert.False(t, log.SentAt.IsZero())
}

// Validates: Requirements 1.2
func TestNotificationTemplate_Fields_AllPresent(t *testing.T) {
	tmpl := fixtures.ValidNotificationTemplate()
	assert.NotEmpty(t, tmpl.TemplateID)
	assert.NotEmpty(t, tmpl.EventType)
	assert.NotEmpty(t, string(tmpl.Channel))
	assert.NotEmpty(t, tmpl.Subject)
	assert.NotEmpty(t, tmpl.BodyTemplate)
}

// Validates: Requirements 1.3
func TestChannel_Validate_ValidChannels(t *testing.T) {
	cases := []struct {
		name    string
		channel domain.Channel
	}{
		{"push", domain.ChannelPush},
		{"email", domain.ChannelEmail},
		{"whatsapp", domain.ChannelWhatsApp},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.channel.Validate()
			require.NoError(t, err)
		})
	}
}

// Validates: Requirements 1.4, 1.5
func TestChannel_Validate_InvalidChannels(t *testing.T) {
	cases := []struct {
		name    string
		channel domain.Channel
	}{
		{"empty_string", domain.Channel("")},
		{"lowercase_push", domain.Channel("push")},
		{"unknown_sms", domain.Channel("SMS")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.channel.Validate()
			require.Error(t, err)
			assert.Contains(t, err.Error(), string(tc.channel))
		})
	}
}

// Validates: Requirements 1.6
func TestRenderTemplate_SubstitutesAllPlaceholders(t *testing.T) {
	tmpl := "Paket {{tracking_number}} sedang dalam perjalanan."
	vars := map[string]string{"tracking_number": "TRK123"}
	result := domain.RenderTemplate(tmpl, vars)
	assert.Contains(t, result, "TRK123")
	assert.NotContains(t, result, "{{")
	assert.NotContains(t, result, "}}")
}

// Validates: Requirements 1.7
func TestRenderTemplate_UnmatchedPlaceholderUnchanged(t *testing.T) {
	tmpl := "Hello {{name}}, your tracking is {{tracking_number}}."
	vars := map[string]string{"tracking_number": "TRK123"}
	result := domain.RenderTemplate(tmpl, vars)
	assert.Contains(t, result, "{{name}}")
	assert.Contains(t, result, "TRK123")
}

// Validates: Requirements 1.8
func TestRenderTemplate_SameInputSameOutput(t *testing.T) {
	tmpl := "Paket {{tracking_number}} sedang dalam perjalanan."
	vars := map[string]string{"tracking_number": "TRK123"}
	result1 := domain.RenderTemplate(tmpl, vars)
	result2 := domain.RenderTemplate(tmpl, vars)
	assert.Equal(t, result1, result2)
}

// Feature: notification-service-unit-tests, Property 1: Invalid channel validation always returns error containing the value
// Validates: Requirements 1.4, 1.5
func TestChannel_Validate_InvalidAlwaysReturnsError(t *testing.T) {
	validChannels := map[string]bool{"PUSH": true, "EMAIL": true, "WHATSAPP": true}
	rapid.Check(t, func(t *rapid.T) {
		ch := rapid.StringN(1, 20, -1).Draw(t, "channel")
		if validChannels[ch] {
			t.Skip()
		}
		err := domain.Channel(ch).Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), ch)
	})
}

// Feature: notification-service-unit-tests, Property 2: RenderTemplate substitutes all present placeholders
// Validates: Requirements 1.6
func TestRenderTemplate_SubstitutesAllPresentPlaceholders(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		key := rapid.StringMatching(`[a-z_]{1,20}`).Draw(t, "key")
		val := rapid.StringN(1, 50, -1).Draw(t, "value")
		tmpl := "{{" + key + "}}"
		vars := map[string]string{key: val}

		result := domain.RenderTemplate(tmpl, vars)
		assert.Equal(t, val, result)
		assert.NotContains(t, result, "{{")
		assert.NotContains(t, result, "}}")
	})
}

// Feature: notification-service-unit-tests, Property 3: RenderTemplate is idempotent
// Validates: Requirements 1.8, 1.11
func TestRenderTemplate_Idempotent(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		tmpl := rapid.StringN(0, 100, -1).Draw(t, "template")
		key := rapid.StringMatching(`[a-z_]{1,20}`).Draw(t, "key")
		val := rapid.StringN(0, 50, -1).Draw(t, "value")
		vars := map[string]string{key: val}

		once := domain.RenderTemplate(tmpl, vars)
		twice := domain.RenderTemplate(once, vars)
		assert.Equal(t, once, twice)
	})
}

// Feature: notification-service-unit-tests, Property 4: NotificationLog and NotificationTemplate JSON round-trip
// Validates: Requirements 1.9, 1.10
func TestNotificationLog_JSON_RoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		channels := []domain.Channel{domain.ChannelPush, domain.ChannelEmail, domain.ChannelWhatsApp}
		statuses := []domain.NotifStatus{domain.NotifStatusSent, domain.NotifStatusFailed, domain.NotifStatusPending}

		original := &domain.NotificationLog{
			NotifID:        rapid.StringMatching(`[a-z0-9-]{4,20}`).Draw(t, "notif_id"),
			UserID:         rapid.StringMatching(`[a-z0-9-]{4,20}`).Draw(t, "user_id"),
			TrackingNumber: rapid.StringMatching(`[A-Z0-9]{4,16}`).Draw(t, "tracking_number"),
			Channel:        channels[rapid.IntRange(0, 2).Draw(t, "channel_idx")],
			TemplateID:     rapid.StringMatching(`[a-z0-9-]{4,20}`).Draw(t, "template_id"),
			Message:        rapid.StringN(1, 100, -1).Draw(t, "message"),
			Status:         statuses[rapid.IntRange(0, 2).Draw(t, "status_idx")],
			SentAt:         time.Unix(rapid.Int64Range(0, 1e9).Draw(t, "sent_at"), 0).UTC(),
		}

		data, err := json.Marshal(original)
		require.NoError(t, err)

		var restored domain.NotificationLog
		require.NoError(t, json.Unmarshal(data, &restored))

		assert.Equal(t, original.NotifID, restored.NotifID)
		assert.Equal(t, original.UserID, restored.UserID)
		assert.Equal(t, original.TrackingNumber, restored.TrackingNumber)
		assert.Equal(t, original.Channel, restored.Channel)
		assert.Equal(t, original.TemplateID, restored.TemplateID)
		assert.Equal(t, original.Message, restored.Message)
		assert.Equal(t, original.Status, restored.Status)
	})
}

func TestNotificationTemplate_JSON_RoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		channels := []domain.Channel{domain.ChannelPush, domain.ChannelEmail, domain.ChannelWhatsApp}

		original := &domain.NotificationTemplate{
			TemplateID:   rapid.StringMatching(`[a-z0-9-]{4,20}`).Draw(t, "template_id"),
			EventType:    rapid.StringMatching(`[A-Z_]{4,20}`).Draw(t, "event_type"),
			Channel:      channels[rapid.IntRange(0, 2).Draw(t, "channel_idx")],
			Subject:      rapid.StringN(1, 100, -1).Draw(t, "subject"),
			BodyTemplate: rapid.StringN(1, 200, -1).Draw(t, "body_template"),
		}

		data, err := json.Marshal(original)
		require.NoError(t, err)

		var restored domain.NotificationTemplate
		require.NoError(t, json.Unmarshal(data, &restored))

		assert.Equal(t, *original, restored)
	})
}
