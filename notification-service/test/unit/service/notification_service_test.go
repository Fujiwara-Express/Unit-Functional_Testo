package service_test

import (
	"context"
	"testing"

	"github.com/notification-service/internal/domain"
	"github.com/notification-service/internal/service"
	"github.com/notification-service/test/mocks"
	"github.com/notification-service/test/unit/fixtures"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"pgregory.net/rapid"
)

// helper to build a service with all mocks
func newTestService(t *testing.T) (
	service.NotificationService,
	*mocks.MockNotificationRepository,
	*mocks.MockFirebaseClient,
	*mocks.MockSendGridClient,
	*mocks.MockWhatsAppClient,
) {
	ctrl := gomock.NewController(t)
	mockRepo := mocks.NewMockNotificationRepository(ctrl)
	mockFB := mocks.NewMockFirebaseClient(ctrl)
	mockSG := mocks.NewMockSendGridClient(ctrl)
	mockWA := mocks.NewMockWhatsAppClient(ctrl)
	svc := service.NewNotificationService(mockRepo, mockFB, mockSG, mockWA)
	return svc, mockRepo, mockFB, mockSG, mockWA
}

// Validates: Requirements 3.1, 3.3
func TestNotificationService_SendNotification_ChannelPush(t *testing.T) {
	svc, mockRepo, mockFB, mockSG, mockWA := newTestService(t)

	tmpl := fixtures.ValidNotificationTemplate()
	tmpl.Channel = domain.ChannelPush
	mockRepo.EXPECT().GetTemplateByID(gomock.Any(), tmpl.TemplateID).Return(tmpl, nil)
	mockFB.EXPECT().SendPush(gomock.Any(), "user-456", gomock.Any()).Return(nil).Times(1)
	mockSG.EXPECT().SendEmail(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	mockWA.EXPECT().SendWhatsApp(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	mockRepo.EXPECT().CreateNotificationLog(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, log *domain.NotificationLog) (string, error) {
			assert.Equal(t, domain.NotifStatusSent, log.Status)
			return "notif-123", nil
		},
	)

	out, err := svc.SendNotification(context.Background(), service.SendNotificationInput{
		UserID:     "user-456",
		Channel:    domain.ChannelPush,
		TemplateID: tmpl.TemplateID,
		Variables:  map[string]string{"tracking_number": "TRK123"},
	})
	require.NoError(t, err)
	assert.NotEmpty(t, out.NotificationID)
	assert.Equal(t, domain.NotifStatusSent, out.Status)
}

// Validates: Requirements 3.4
func TestNotificationService_SendNotification_ChannelEmail(t *testing.T) {
	svc, mockRepo, mockFB, mockSG, mockWA := newTestService(t)

	tmpl := fixtures.ValidNotificationTemplate()
	tmpl.Channel = domain.ChannelEmail
	mockRepo.EXPECT().GetTemplateByID(gomock.Any(), tmpl.TemplateID).Return(tmpl, nil)
	mockSG.EXPECT().SendEmail(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)
	mockFB.EXPECT().SendPush(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	mockWA.EXPECT().SendWhatsApp(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	mockRepo.EXPECT().CreateNotificationLog(gomock.Any(), gomock.Any()).Return("notif-456", nil)

	out, err := svc.SendNotification(context.Background(), service.SendNotificationInput{
		UserID:     "user-456",
		Channel:    domain.ChannelEmail,
		TemplateID: tmpl.TemplateID,
		Variables:  map[string]string{},
	})
	require.NoError(t, err)
	assert.Equal(t, domain.ChannelEmail, out.Channel)
}

// Validates: Requirements 3.5
func TestNotificationService_SendNotification_ChannelWhatsApp(t *testing.T) {
	svc, mockRepo, mockFB, mockSG, mockWA := newTestService(t)

	tmpl := fixtures.ValidNotificationTemplate()
	tmpl.Channel = domain.ChannelWhatsApp
	mockRepo.EXPECT().GetTemplateByID(gomock.Any(), tmpl.TemplateID).Return(tmpl, nil)
	mockWA.EXPECT().SendWhatsApp(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)
	mockFB.EXPECT().SendPush(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	mockSG.EXPECT().SendEmail(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	mockRepo.EXPECT().CreateNotificationLog(gomock.Any(), gomock.Any()).Return("notif-789", nil)

	out, err := svc.SendNotification(context.Background(), service.SendNotificationInput{
		UserID:     "user-456",
		Channel:    domain.ChannelWhatsApp,
		TemplateID: tmpl.TemplateID,
		Variables:  map[string]string{},
	})
	require.NoError(t, err)
	assert.Equal(t, domain.ChannelWhatsApp, out.Channel)
}

// Validates: Requirements 3.2
func TestNotificationService_SendNotification_ProviderError(t *testing.T) {
	svc, mockRepo, mockFB, _, _ := newTestService(t)

	tmpl := fixtures.ValidNotificationTemplate()
	tmpl.Channel = domain.ChannelPush
	mockRepo.EXPECT().GetTemplateByID(gomock.Any(), tmpl.TemplateID).Return(tmpl, nil)
	mockFB.EXPECT().SendPush(gomock.Any(), gomock.Any(), gomock.Any()).Return(domain.ErrServiceUnavailable)
	mockRepo.EXPECT().CreateNotificationLog(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, log *domain.NotificationLog) (string, error) {
			assert.Equal(t, domain.NotifStatusFailed, log.Status)
			return "notif-fail", nil
		},
	)

	_, err := svc.SendNotification(context.Background(), service.SendNotificationInput{
		UserID:     "user-456",
		Channel:    domain.ChannelPush,
		TemplateID: tmpl.TemplateID,
		Variables:  map[string]string{},
	})
	require.Error(t, err)
}

// Validates: Requirements 3.6
func TestNotificationService_SendNotification_TemplateNotFound(t *testing.T) {
	svc, mockRepo, mockFB, mockSG, mockWA := newTestService(t)

	mockRepo.EXPECT().GetTemplateByID(gomock.Any(), "nonexistent").Return(nil, domain.ErrNotFound)
	mockFB.EXPECT().SendPush(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	mockSG.EXPECT().SendEmail(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	mockWA.EXPECT().SendWhatsApp(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	_, err := svc.SendNotification(context.Background(), service.SendNotificationInput{
		UserID:     "user-456",
		Channel:    domain.ChannelPush,
		TemplateID: "nonexistent",
		Variables:  map[string]string{},
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

// Validates: Requirements 3.7
func TestNotificationService_ListTemplates_ReturnsAll(t *testing.T) {
	svc, mockRepo, _, _, _ := newTestService(t)

	expected := []*domain.NotificationTemplate{fixtures.ValidNotificationTemplate()}
	mockRepo.EXPECT().ListTemplates(gomock.Any()).Return(expected, nil)

	got, err := svc.ListTemplates(context.Background())
	require.NoError(t, err)
	assert.Equal(t, expected, got)
}

// Validates: Requirements 3.8
func TestNotificationService_CreateTemplate_ValidInput(t *testing.T) {
	svc, mockRepo, _, _, _ := newTestService(t)

	mockRepo.EXPECT().CreateTemplate(gomock.Any(), gomock.Any()).Return("tmpl-new", nil).Times(1)

	out, err := svc.CreateTemplate(context.Background(), service.CreateTemplateInput{
		EventType:    "OUT_FOR_DELIVERY",
		Channel:      domain.ChannelPush,
		Subject:      "Package Update",
		BodyTemplate: "Your package {{tracking_number}} is on the way.",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, out.TemplateID)
	assert.Equal(t, "CREATED", out.Status)
}

// Validates: Requirements 3.9
func TestNotificationService_UpdateTemplate_ValidInput(t *testing.T) {
	svc, mockRepo, _, _, _ := newTestService(t)

	tmpl := fixtures.ValidNotificationTemplate()
	mockRepo.EXPECT().GetTemplateByID(gomock.Any(), tmpl.TemplateID).Return(tmpl, nil)
	mockRepo.EXPECT().UpdateTemplate(gomock.Any(), tmpl.TemplateID, "New Subject", "New body").Return(nil)

	out, err := svc.UpdateTemplate(context.Background(), tmpl.TemplateID, service.UpdateTemplateInput{
		Subject:      "New Subject",
		BodyTemplate: "New body",
	})
	require.NoError(t, err)
	assert.Equal(t, tmpl.TemplateID, out.TemplateID)
	assert.Equal(t, "UPDATED", out.Status)
}

// Validates: Requirements 3.10
func TestNotificationService_UpdateTemplate_NotFound(t *testing.T) {
	svc, mockRepo, _, _, _ := newTestService(t)

	mockRepo.EXPECT().GetTemplateByID(gomock.Any(), "nonexistent").Return(nil, domain.ErrNotFound)
	mockRepo.EXPECT().UpdateTemplate(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	_, err := svc.UpdateTemplate(context.Background(), "nonexistent", service.UpdateTemplateInput{
		Subject:      "Subject",
		BodyTemplate: "Body",
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

// Feature: notification-service-unit-tests, Property 5: SendNotification always calls CreateNotificationLog exactly once
// Validates: Requirements 3.1, 3.2, 3.11
func TestNotificationService_SendNotification_AlwaysLogsOnce(t *testing.T) {
	channels := []domain.Channel{domain.ChannelPush, domain.ChannelEmail, domain.ChannelWhatsApp}
	providerErrors := []error{nil, domain.ErrServiceUnavailable}

	rapid.Check(t, func(rt *rapid.T) {
		ctrl := gomock.NewController(t)
		mockRepo := mocks.NewMockNotificationRepository(ctrl)
		mockFB := mocks.NewMockFirebaseClient(ctrl)
		mockSG := mocks.NewMockSendGridClient(ctrl)
		mockWA := mocks.NewMockWhatsAppClient(ctrl)
		svc := service.NewNotificationService(mockRepo, mockFB, mockSG, mockWA)

		chIdx := rapid.IntRange(0, 2).Draw(rt, "channel_idx")
		ch := channels[chIdx]
		provErrIdx := rapid.IntRange(0, 1).Draw(rt, "provider_err_idx")
		provErr := providerErrors[provErrIdx]

		tmpl := fixtures.ValidNotificationTemplate()
		tmpl.Channel = ch

		mockRepo.EXPECT().GetTemplateByID(gomock.Any(), tmpl.TemplateID).Return(tmpl, nil)

		switch ch {
		case domain.ChannelPush:
			mockFB.EXPECT().SendPush(gomock.Any(), gomock.Any(), gomock.Any()).Return(provErr)
		case domain.ChannelEmail:
			mockSG.EXPECT().SendEmail(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(provErr)
		case domain.ChannelWhatsApp:
			mockWA.EXPECT().SendWhatsApp(gomock.Any(), gomock.Any(), gomock.Any()).Return(provErr)
		}

		logCallCount := 0
		mockRepo.EXPECT().CreateNotificationLog(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, log *domain.NotificationLog) (string, error) {
				logCallCount++
				if provErr != nil {
					assert.Equal(t, domain.NotifStatusFailed, log.Status)
				} else {
					assert.Equal(t, domain.NotifStatusSent, log.Status)
				}
				return "notif-id", nil
			},
		).Times(1)

		svc.SendNotification(context.Background(), service.SendNotificationInput{
			UserID:     "user-456",
			Channel:    ch,
			TemplateID: tmpl.TemplateID,
			Variables:  map[string]string{},
		})

		assert.Equal(t, 1, logCallCount, "CreateNotificationLog must be called exactly once")
	})
}

// Feature: notification-service-unit-tests, Property 8: Service CreateTemplate returns non-empty template_id for any valid input
// Validates: Requirements 3.8
func TestNotificationService_CreateTemplate_AnyValidInput(t *testing.T) {
	channels := []domain.Channel{domain.ChannelPush, domain.ChannelEmail, domain.ChannelWhatsApp}

	rapid.Check(t, func(rt *rapid.T) {
		ctrl := gomock.NewController(t)
		mockRepo := mocks.NewMockNotificationRepository(ctrl)
		mockFB := mocks.NewMockFirebaseClient(ctrl)
		mockSG := mocks.NewMockSendGridClient(ctrl)
		mockWA := mocks.NewMockWhatsAppClient(ctrl)
		svc := service.NewNotificationService(mockRepo, mockFB, mockSG, mockWA)

		generatedID := rapid.StringMatching(`[a-z0-9-]{4,20}`).Draw(rt, "template_id")
		mockRepo.EXPECT().CreateTemplate(gomock.Any(), gomock.Any()).Return(generatedID, nil).Times(1)

		input := service.CreateTemplateInput{
			EventType:    rapid.StringMatching(`[A-Z_]{4,20}`).Draw(rt, "event_type"),
			Channel:      channels[rapid.IntRange(0, 2).Draw(rt, "channel_idx")],
			Subject:      rapid.StringN(1, 100, -1).Draw(rt, "subject"),
			BodyTemplate: rapid.StringN(1, 200, -1).Draw(rt, "body_template"),
		}

		out, err := svc.CreateTemplate(context.Background(), input)
		require.NoError(t, err)
		assert.NotEmpty(t, out.TemplateID)
		assert.Equal(t, "CREATED", out.Status)
	})
}

// Additional error-path tests to improve coverage

func TestNotificationService_SendNotification_LogError(t *testing.T) {
	svc, mockRepo, mockFB, _, _ := newTestService(t)

	tmpl := fixtures.ValidNotificationTemplate()
	tmpl.Channel = domain.ChannelPush
	mockRepo.EXPECT().GetTemplateByID(gomock.Any(), tmpl.TemplateID).Return(tmpl, nil)
	mockFB.EXPECT().SendPush(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
	mockRepo.EXPECT().CreateNotificationLog(gomock.Any(), gomock.Any()).Return("", domain.ErrServiceUnavailable)

	_, err := svc.SendNotification(context.Background(), service.SendNotificationInput{
		UserID:     "user-456",
		Channel:    domain.ChannelPush,
		TemplateID: tmpl.TemplateID,
		Variables:  map[string]string{},
	})
	require.Error(t, err)
}

func TestNotificationService_ListTemplates_Error(t *testing.T) {
	svc, mockRepo, _, _, _ := newTestService(t)

	mockRepo.EXPECT().ListTemplates(gomock.Any()).Return(nil, domain.ErrServiceUnavailable)

	_, err := svc.ListTemplates(context.Background())
	require.Error(t, err)
}

func TestNotificationService_CreateTemplate_RepoError(t *testing.T) {
	svc, mockRepo, _, _, _ := newTestService(t)

	mockRepo.EXPECT().CreateTemplate(gomock.Any(), gomock.Any()).Return("", domain.ErrServiceUnavailable)

	_, err := svc.CreateTemplate(context.Background(), service.CreateTemplateInput{
		EventType:    "OUT_FOR_DELIVERY",
		Channel:      domain.ChannelPush,
		Subject:      "Subject",
		BodyTemplate: "Body",
	})
	require.Error(t, err)
}

func TestNotificationService_UpdateTemplate_RepoError(t *testing.T) {
	svc, mockRepo, _, _, _ := newTestService(t)

	tmpl := fixtures.ValidNotificationTemplate()
	mockRepo.EXPECT().GetTemplateByID(gomock.Any(), tmpl.TemplateID).Return(tmpl, nil)
	mockRepo.EXPECT().UpdateTemplate(gomock.Any(), tmpl.TemplateID, gomock.Any(), gomock.Any()).Return(domain.ErrServiceUnavailable)

	_, err := svc.UpdateTemplate(context.Background(), tmpl.TemplateID, service.UpdateTemplateInput{
		Subject:      "New Subject",
		BodyTemplate: "New body",
	})
	require.Error(t, err)
}
