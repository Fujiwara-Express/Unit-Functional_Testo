package service_test

import (
	"context"
	"testing"

	"github.com/pickup-service/internal/domain"
	"github.com/pickup-service/internal/service"
	"github.com/pickup-service/test/mocks"
	"github.com/pickup-service/test/unit/fixtures"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"pgregory.net/rapid"
)

// helper to build a service with all mocks
func newTestService(t *testing.T) (
	service.PickupService,
	*mocks.MockPickupRepository,
	*mocks.MockDeliveryClient,
	*mocks.MockTrackingClient,
	*mocks.MockNotificationClient,
) {
	ctrl := gomock.NewController(t)
	mockRepo := mocks.NewMockPickupRepository(ctrl)
	mockDelivery := mocks.NewMockDeliveryClient(ctrl)
	mockTracking := mocks.NewMockTrackingClient(ctrl)
	mockNotification := mocks.NewMockNotificationClient(ctrl)
	svc := service.NewPickupService(mockRepo, mockDelivery, mockTracking, mockNotification)
	return svc, mockRepo, mockDelivery, mockTracking, mockNotification
}

// 10.1
func TestPickupService_RequestPickup_ValidInput(t *testing.T) {
	svc, mockRepo, _, _, _ := newTestService(t)

	mockRepo.EXPECT().
		CreatePickup(gomock.Any(), gomock.Any()).
		Return("pickup-123", nil)

	out, err := svc.RequestPickup(context.Background(), service.RequestPickupInput{
		OrderID:        "order-456",
		UserID:         "user-789",
		PickupAddress:  "123 Main St",
		PickupCityCode: "JKT",
		ContactName:    "John Doe",
		ContactPhone:   "+62812345678",
	})

	require.NoError(t, err)
	assert.NotEmpty(t, out.PickupID)
	assert.Equal(t, domain.StatusScheduled, out.Status)
}

// 10.2
func TestPickupService_AssignCourier_ValidInput(t *testing.T) {
	svc, mockRepo, _, _, _ := newTestService(t)

	existing := fixtures.PickupWithStatus(domain.StatusScheduled)
	mockRepo.EXPECT().GetPickupByID(gomock.Any(), existing.PickupID).Return(existing, nil)
	mockRepo.EXPECT().UpdatePickup(gomock.Any(), gomock.Any()).Return(nil)

	out, err := svc.AssignCourier(context.Background(), existing.PickupID, "courier-456")

	require.NoError(t, err)
	assert.Equal(t, domain.StatusAssigned, out.Status)
}

// 10.3
func TestPickupService_AssignCourier_NotFound(t *testing.T) {
	svc, mockRepo, _, _, _ := newTestService(t)

	mockRepo.EXPECT().GetPickupByID(gomock.Any(), "nonexistent").Return(nil, domain.ErrNotFound)
	mockRepo.EXPECT().UpdatePickup(gomock.Any(), gomock.Any()).Times(0)

	_, err := svc.AssignCourier(context.Background(), "nonexistent", "courier-456")

	assert.ErrorIs(t, err, domain.ErrNotFound)
}

// 10.4
func TestPickupService_UpdatePickupStatus_PickedUp(t *testing.T) {
	svc, mockRepo, _, mockTracking, _ := newTestService(t)

	existing := fixtures.PickupWithStatus(domain.StatusAssigned)
	mockRepo.EXPECT().GetPickupByID(gomock.Any(), existing.PickupID).Return(existing, nil)
	mockRepo.EXPECT().UpdatePickup(gomock.Any(), gomock.Any()).Return(nil)
	mockTracking.EXPECT().
		PublishPickedUpEvent(gomock.Any(), existing.PickupID, existing.OrderID, gomock.Any()).
		Return(nil).
		Times(1)

	_, err := svc.UpdatePickupStatus(context.Background(), existing.PickupID, domain.StatusPickedUp)

	require.NoError(t, err)
}

// 10.5
func TestPickupService_UpdatePickupStatus_FailedAttempt(t *testing.T) {
	svc, mockRepo, _, _, mockNotification := newTestService(t)

	existing := fixtures.PickupWithStatus(domain.StatusAssigned)
	existing.AttemptCount = 0

	var capturedPickup *domain.Pickup
	mockRepo.EXPECT().GetPickupByID(gomock.Any(), existing.PickupID).Return(existing, nil)
	mockRepo.EXPECT().UpdatePickup(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, p *domain.Pickup) error {
		capturedPickup = p
		return nil
	})
	mockNotification.EXPECT().
		NotifyCourierEnRoute(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil)

	_, err := svc.UpdatePickupStatus(context.Background(), existing.PickupID, domain.StatusFailedAttempt)

	require.NoError(t, err)
	if capturedPickup != nil {
		assert.Equal(t, 1, capturedPickup.AttemptCount)
	}
}

// 10.6
func TestPickupService_UpdatePickupStatus_InvalidTransition(t *testing.T) {
	svc, mockRepo, _, _, _ := newTestService(t)

	existing := fixtures.PickupWithStatus(domain.StatusPickedUp)
	mockRepo.EXPECT().GetPickupByID(gomock.Any(), existing.PickupID).Return(existing, nil)
	mockRepo.EXPECT().UpdatePickup(gomock.Any(), gomock.Any()).Times(0)

	_, err := svc.UpdatePickupStatus(context.Background(), existing.PickupID, domain.StatusAssigned)

	assert.Error(t, err)
}

// 10.7
func TestPickupService_CancelPickup_Scheduled(t *testing.T) {
	svc, mockRepo, _, _, _ := newTestService(t)

	existing := fixtures.PickupWithStatus(domain.StatusScheduled)
	mockRepo.EXPECT().GetPickupByID(gomock.Any(), existing.PickupID).Return(existing, nil)
	mockRepo.EXPECT().UpdatePickup(gomock.Any(), gomock.Any()).Return(nil)

	out, err := svc.CancelPickup(context.Background(), existing.PickupID)

	require.NoError(t, err)
	assert.Equal(t, domain.StatusCancelled, out.Status)
}

// 10.8
func TestPickupService_CancelPickup_NonScheduled(t *testing.T) {
	cases := []struct {
		name   string
		status domain.Status
	}{
		{"assigned", domain.StatusAssigned},
		{"picked_up", domain.StatusPickedUp},
		{"failed_attempt", domain.StatusFailedAttempt},
		{"cancelled", domain.StatusCancelled},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc, mockRepo, _, _, _ := newTestService(t)

			existing := fixtures.PickupWithStatus(tc.status)
			mockRepo.EXPECT().GetPickupByID(gomock.Any(), existing.PickupID).Return(existing, nil)
			mockRepo.EXPECT().UpdatePickup(gomock.Any(), gomock.Any()).Times(0)

			_, err := svc.CancelPickup(context.Background(), existing.PickupID)

			assert.ErrorIs(t, err, domain.ErrConflict)
		})
	}
}

// 10.9
func TestPickupService_GetPickup_ValidInput(t *testing.T) {
	svc, mockRepo, _, _, _ := newTestService(t)

	expected := fixtures.ValidPickup()
	mockRepo.EXPECT().GetPickupByID(gomock.Any(), expected.PickupID).Return(expected, nil)

	got, err := svc.GetPickup(context.Background(), expected.PickupID)

	require.NoError(t, err)
	assert.Equal(t, expected, got)
}

func TestPickupService_ListPickups_WithFilters(t *testing.T) {
	svc, mockRepo, _, _, _ := newTestService(t)

	expected := []*domain.Pickup{fixtures.ValidPickup()}
	mockRepo.EXPECT().ListPickups(gomock.Any(), gomock.Any()).Return(expected, nil)

	got, err := svc.ListPickups(context.Background(), service.ListFilters{Status: "SCHEDULED"})

	require.NoError(t, err)
	assert.Equal(t, expected, got)
}

// 10.10
// Feature: pickup-service-unit-tests, Property 9: Service RequestPickup calls CreatePickup once and returns SCHEDULED
func TestPickupService_RequestPickup_AnyValidInput(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		ctrl := gomock.NewController(t)
		mockRepo := mocks.NewMockPickupRepository(ctrl)
		mockDelivery := mocks.NewMockDeliveryClient(ctrl)
		mockTracking := mocks.NewMockTrackingClient(ctrl)
		mockNotification := mocks.NewMockNotificationClient(ctrl)
		svc := service.NewPickupService(mockRepo, mockDelivery, mockTracking, mockNotification)

		generatedID := rapid.StringMatching(`[a-z0-9-]{8,36}`).Draw(rt, "pickup_id")

		input := service.RequestPickupInput{
			OrderID:        rapid.StringMatching(`[a-z0-9-]{8,36}`).Draw(rt, "order_id"),
			UserID:         rapid.StringMatching(`[a-z0-9-]{8,36}`).Draw(rt, "user_id"),
			PickupAddress:  rapid.StringN(5, 100, -1).Draw(rt, "address"),
			PickupCityCode: rapid.StringMatching(`[A-Z]{3}`).Draw(rt, "city_code"),
			ContactName:    rapid.StringN(2, 50, -1).Draw(rt, "contact_name"),
			ContactPhone:   rapid.StringMatching(`\+[0-9]{10,15}`).Draw(rt, "contact_phone"),
		}

		mockRepo.EXPECT().
			CreatePickup(gomock.Any(), gomock.Any()).
			Return(generatedID, nil).
			Times(1)

		out, err := svc.RequestPickup(context.Background(), input)

		require.NoError(t, err)
		assert.NotEmpty(t, out.PickupID)
		assert.Equal(t, domain.StatusScheduled, out.Status)
	})
}

// 10.11
// Feature: pickup-service-unit-tests, Property 10: Service CancelPickup rejects non-SCHEDULED pickups
func TestPickupService_CancelPickup_RejectsNonScheduled(t *testing.T) {
	nonScheduled := []domain.Status{
		domain.StatusAssigned,
		domain.StatusPickedUp,
		domain.StatusFailedAttempt,
		domain.StatusCancelled,
	}

	rapid.Check(t, func(rt *rapid.T) {
		ctrl := gomock.NewController(t)
		mockRepo := mocks.NewMockPickupRepository(ctrl)
		mockDelivery := mocks.NewMockDeliveryClient(ctrl)
		mockTracking := mocks.NewMockTrackingClient(ctrl)
		mockNotification := mocks.NewMockNotificationClient(ctrl)
		svc := service.NewPickupService(mockRepo, mockDelivery, mockTracking, mockNotification)

		idx := rapid.IntRange(0, len(nonScheduled)-1).Draw(rt, "status_idx")
		status := nonScheduled[idx]

		existing := fixtures.PickupWithStatus(status)
		mockRepo.EXPECT().GetPickupByID(gomock.Any(), existing.PickupID).Return(existing, nil)
		mockRepo.EXPECT().UpdatePickup(gomock.Any(), gomock.Any()).Times(0)

		_, err := svc.CancelPickup(context.Background(), existing.PickupID)

		assert.Error(t, err)
	})
}

// --- Additional error-path tests to improve coverage ---

func TestPickupService_RequestPickup_RepoError(t *testing.T) {
	svc, mockRepo, _, _, _ := newTestService(t)

	mockRepo.EXPECT().
		CreatePickup(gomock.Any(), gomock.Any()).
		Return("", domain.ErrServiceUnavailable)

	_, err := svc.RequestPickup(context.Background(), service.RequestPickupInput{
		OrderID:        "order-1",
		UserID:         "user-1",
		PickupAddress:  "123 Main St",
		PickupCityCode: "JKT",
		ContactName:    "John",
		ContactPhone:   "+628123",
	})

	assert.Error(t, err)
}

func TestPickupService_AssignCourier_UpdateError(t *testing.T) {
	svc, mockRepo, _, _, _ := newTestService(t)

	existing := fixtures.PickupWithStatus(domain.StatusScheduled)
	mockRepo.EXPECT().GetPickupByID(gomock.Any(), existing.PickupID).Return(existing, nil)
	mockRepo.EXPECT().UpdatePickup(gomock.Any(), gomock.Any()).Return(domain.ErrServiceUnavailable)

	_, err := svc.AssignCourier(context.Background(), existing.PickupID, "courier-1")

	assert.Error(t, err)
}

func TestPickupService_AssignCourier_InvalidTransition(t *testing.T) {
	svc, mockRepo, _, _, _ := newTestService(t)

	existing := fixtures.PickupWithStatus(domain.StatusPickedUp)
	mockRepo.EXPECT().GetPickupByID(gomock.Any(), existing.PickupID).Return(existing, nil)
	mockRepo.EXPECT().UpdatePickup(gomock.Any(), gomock.Any()).Times(0)

	_, err := svc.AssignCourier(context.Background(), existing.PickupID, "courier-1")

	assert.Error(t, err)
}

func TestPickupService_UpdatePickupStatus_UpdateError(t *testing.T) {
	svc, mockRepo, _, _, _ := newTestService(t)

	existing := fixtures.PickupWithStatus(domain.StatusScheduled)
	mockRepo.EXPECT().GetPickupByID(gomock.Any(), existing.PickupID).Return(existing, nil)
	mockRepo.EXPECT().UpdatePickup(gomock.Any(), gomock.Any()).Return(domain.ErrServiceUnavailable)

	_, err := svc.UpdatePickupStatus(context.Background(), existing.PickupID, domain.StatusAssigned)

	assert.Error(t, err)
}

func TestPickupService_UpdatePickupStatus_TrackingError(t *testing.T) {
	svc, mockRepo, _, mockTracking, _ := newTestService(t)

	existing := fixtures.PickupWithStatus(domain.StatusAssigned)
	mockRepo.EXPECT().GetPickupByID(gomock.Any(), existing.PickupID).Return(existing, nil)
	mockRepo.EXPECT().UpdatePickup(gomock.Any(), gomock.Any()).Return(nil)
	mockTracking.EXPECT().
		PublishPickedUpEvent(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(domain.ErrServiceUnavailable)

	_, err := svc.UpdatePickupStatus(context.Background(), existing.PickupID, domain.StatusPickedUp)

	assert.Error(t, err)
}

func TestPickupService_UpdatePickupStatus_NotificationError(t *testing.T) {
	svc, mockRepo, _, _, mockNotification := newTestService(t)

	existing := fixtures.PickupWithStatus(domain.StatusAssigned)
	mockRepo.EXPECT().GetPickupByID(gomock.Any(), existing.PickupID).Return(existing, nil)
	mockRepo.EXPECT().UpdatePickup(gomock.Any(), gomock.Any()).Return(nil)
	mockNotification.EXPECT().
		NotifyCourierEnRoute(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(domain.ErrServiceUnavailable)

	_, err := svc.UpdatePickupStatus(context.Background(), existing.PickupID, domain.StatusFailedAttempt)

	assert.Error(t, err)
}

func TestPickupService_CancelPickup_UpdateError(t *testing.T) {
	svc, mockRepo, _, _, _ := newTestService(t)

	existing := fixtures.PickupWithStatus(domain.StatusScheduled)
	mockRepo.EXPECT().GetPickupByID(gomock.Any(), existing.PickupID).Return(existing, nil)
	mockRepo.EXPECT().UpdatePickup(gomock.Any(), gomock.Any()).Return(domain.ErrServiceUnavailable)

	_, err := svc.CancelPickup(context.Background(), existing.PickupID)

	assert.Error(t, err)
}

func TestPickupService_GetPickup_NotFound(t *testing.T) {
	svc, mockRepo, _, _, _ := newTestService(t)

	mockRepo.EXPECT().GetPickupByID(gomock.Any(), "missing").Return(nil, domain.ErrNotFound)

	_, err := svc.GetPickup(context.Background(), "missing")

	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestPickupService_ListPickups_RepoError(t *testing.T) {
	svc, mockRepo, _, _, _ := newTestService(t)

	mockRepo.EXPECT().ListPickups(gomock.Any(), gomock.Any()).Return(nil, domain.ErrServiceUnavailable)

	_, err := svc.ListPickups(context.Background(), service.ListFilters{})

	assert.Error(t, err)
}
