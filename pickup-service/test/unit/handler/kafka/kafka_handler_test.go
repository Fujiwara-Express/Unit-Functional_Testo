package kafka_test

import (
	"encoding/json"
	"os"
	"testing"

	kafkahandler "github.com/pickup-service/internal/handler/kafka"
	"github.com/pickup-service/internal/service"
	"github.com/pickup-service/test/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

// --- 13.5: Kafka handler tests ---

func TestKafkaHandler_HandleOrderCreatedEvent_ValidPayload(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockSvc := mocks.NewMockPickupService(ctrl)

	handler := kafkahandler.NewOrderCreatedHandler(mockSvc)

	event := kafkahandler.OrderCreatedEvent{
		OrderID:        "order-123",
		UserID:         "user-456",
		PickupAddress:  "123 Main St",
		PickupCityCode: "JKT",
		ContactName:    "Jane Doe",
		ContactPhone:   "+62812345678",
	}
	payload, err := json.Marshal(event)
	require.NoError(t, err)

	expectedInput := service.RequestPickupInput{
		OrderID:        event.OrderID,
		UserID:         event.UserID,
		PickupAddress:  event.PickupAddress,
		PickupCityCode: event.PickupCityCode,
		ContactName:    event.ContactName,
		ContactPhone:   event.ContactPhone,
	}

	mockSvc.EXPECT().
		RequestPickup(gomock.Any(), expectedInput).
		Return(&service.RequestPickupOutput{PickupID: "pickup-789"}, nil).
		Times(1)

	err = handler.HandleOrderCreatedEvent(payload)
	assert.NoError(t, err)
}

func TestKafkaHandler_HandleOrderCreatedEvent_MalformedJSON(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockSvc := mocks.NewMockPickupService(ctrl)

	// Service should NEVER be called for malformed JSON
	mockSvc.EXPECT().RequestPickup(gomock.Any(), gomock.Any()).Times(0)

	handler := kafkahandler.NewOrderCreatedHandler(mockSvc)

	err := handler.HandleOrderCreatedEvent([]byte("not-json"))
	assert.Error(t, err, "expected an error for malformed JSON input")
}
