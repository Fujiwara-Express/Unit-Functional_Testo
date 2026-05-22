package kafka_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/notification-service/internal/domain"
	kafkahandler "github.com/notification-service/internal/handler/kafka"
	"github.com/notification-service/internal/service"
	"github.com/notification-service/test/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"pgregory.net/rapid"
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

// Validates: Requirements 6.1
func TestKafkaHandler_HandleTrackingStatusUpdated_ValidPayload(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockSvc := mocks.NewMockNotificationService(ctrl)

	handler := kafkahandler.NewTrackingStatusHandler(mockSvc)

	event := kafkahandler.TrackingStatusUpdatedEvent{
		UserID:         "user-123",
		TrackingNumber: "TRK456",
		EventType:      "OUT_FOR_DELIVERY",
	}
	payload, err := json.Marshal(event)
	require.NoError(t, err)

	mockSvc.EXPECT().
		SendNotification(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ interface{}, req service.SendNotificationInput) (*service.SendNotificationOutput, error) {
			assert.Equal(t, event.UserID, req.UserID)
			assert.Equal(t, domain.ChannelPush, req.Channel)
			assert.Equal(t, event.EventType, req.TemplateID)
			assert.Equal(t, event.TrackingNumber, req.Variables["tracking_number"])
			return &service.SendNotificationOutput{NotificationID: "notif-1"}, nil
		}).
		Times(1)

	err = handler.HandleTrackingStatusUpdated(payload)
	assert.NoError(t, err)
}

// Validates: Requirements 6.2
func TestKafkaHandler_HandleTrackingStatusUpdated_MalformedJSON(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockSvc := mocks.NewMockNotificationService(ctrl)

	mockSvc.EXPECT().SendNotification(gomock.Any(), gomock.Any()).Times(0)

	handler := kafkahandler.NewTrackingStatusHandler(mockSvc)

	err := handler.HandleTrackingStatusUpdated([]byte("not-json"))
	assert.Error(t, err, "expected an error for malformed JSON input")
}

// Validates: Requirements 6.3
func TestKafkaHandler_HandleTrackingStatusUpdated_ServiceError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockSvc := mocks.NewMockNotificationService(ctrl)

	mockSvc.EXPECT().
		SendNotification(gomock.Any(), gomock.Any()).
		Return(nil, domain.ErrServiceUnavailable)

	handler := kafkahandler.NewTrackingStatusHandler(mockSvc)

	event := kafkahandler.TrackingStatusUpdatedEvent{
		UserID:         "user-123",
		TrackingNumber: "TRK456",
		EventType:      "OUT_FOR_DELIVERY",
	}
	payload, err := json.Marshal(event)
	require.NoError(t, err)

	err = handler.HandleTrackingStatusUpdated(payload)
	assert.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrServiceUnavailable)
}

// Feature: notification-service-unit-tests, Property 14: Kafka handler extracts correct parameters for any valid event payload
// Validates: Requirements 6.1
func TestKafkaHandler_ExtractsCorrectParameters(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		ctrl := gomock.NewController(t)
		mockSvc := mocks.NewMockNotificationService(ctrl)
		handler := kafkahandler.NewTrackingStatusHandler(mockSvc)

		event := kafkahandler.TrackingStatusUpdatedEvent{
			UserID:         rapid.StringMatching(`[a-z0-9-]{4,20}`).Draw(rt, "user_id"),
			TrackingNumber: rapid.StringMatching(`[A-Z0-9]{4,16}`).Draw(rt, "tracking_number"),
			EventType:      rapid.StringMatching(`[A-Z_]{4,20}`).Draw(rt, "event_type"),
		}
		payload, err := json.Marshal(event)
		require.NoError(t, err)

		mockSvc.EXPECT().
			SendNotification(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ interface{}, req service.SendNotificationInput) (*service.SendNotificationOutput, error) {
				assert.Equal(t, event.UserID, req.UserID)
				assert.Equal(t, event.EventType, req.TemplateID)
				assert.Equal(t, event.TrackingNumber, req.Variables["tracking_number"])
				return &service.SendNotificationOutput{NotificationID: "notif-1"}, nil
			}).
			Times(1)

		err = handler.HandleTrackingStatusUpdated(payload)
		assert.NoError(t, err)
	})
}
