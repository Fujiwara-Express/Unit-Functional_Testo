package service_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/fujiwara-express/warehouse-service/internal/domain"
	"github.com/fujiwara-express/warehouse-service/internal/service"
	"github.com/fujiwara-express/warehouse-service/mocks"
)

// ── ReceiveItem ──────────────────────────────────────────────────────────────

// TestReceiveItem_NewItem verifies that a brand-new item (not yet in the repo)
// is saved directly without any quantity merge.
func TestReceiveItem_NewItem(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockWarehouseRepository(ctrl)

	newItem := domain.Item{ID: "BRG-001", Name: "Laptop", Quantity: 10, Location: "Rak-A1"}

	// GetItemByID returns not-found → item is new
	mockRepo.EXPECT().
		GetItemByID(gomock.Any(), "BRG-001").
		Return(domain.Item{}, domain.ErrItemNotFound).
		Times(1)

	// SaveItem is called once with the original item
	mockRepo.EXPECT().
		SaveItem(gomock.Any(), newItem).
		Return(nil).
		Times(1)

	svc := service.NewWarehouseService(mockRepo)
	err := svc.ReceiveItem(context.Background(), newItem)
	require.NoError(t, err)
}

// TestReceiveItem_ExistingItem_QuantityMerged verifies that receiving stock for
// an existing item adds the new quantity on top of the existing quantity.
func TestReceiveItem_ExistingItem_QuantityMerged(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockWarehouseRepository(ctrl)

	existing := domain.Item{ID: "BRG-001", Name: "Laptop", Quantity: 10, Location: "Rak-A1"}
	incoming := domain.Item{ID: "BRG-001", Name: "Laptop", Quantity: 5, Location: ""}

	mockRepo.EXPECT().
		GetItemByID(gomock.Any(), "BRG-001").
		Return(existing, nil).
		Times(1)

	// SaveItem must be called with merged quantity (10 + 5 = 15), location unchanged
	expected := domain.Item{ID: "BRG-001", Name: "Laptop", Quantity: 15, Location: "Rak-A1"}
	mockRepo.EXPECT().
		SaveItem(gomock.Any(), expected).
		Return(nil).
		Times(1)

	svc := service.NewWarehouseService(mockRepo)
	err := svc.ReceiveItem(context.Background(), incoming)
	require.NoError(t, err)
}

// TestReceiveItem_ExistingItem_LocationUpdated verifies that a non-empty location
// in the incoming item overwrites the existing location.
func TestReceiveItem_ExistingItem_LocationUpdated(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockWarehouseRepository(ctrl)

	existing := domain.Item{ID: "BRG-002", Name: "Mouse", Quantity: 20, Location: "Rak-B1"}
	incoming := domain.Item{ID: "BRG-002", Name: "Mouse", Quantity: 10, Location: "Rak-C3"}

	mockRepo.EXPECT().
		GetItemByID(gomock.Any(), "BRG-002").
		Return(existing, nil).
		Times(1)

	// Location must be updated to the new value
	expected := domain.Item{ID: "BRG-002", Name: "Mouse", Quantity: 30, Location: "Rak-C3"}
	mockRepo.EXPECT().
		SaveItem(gomock.Any(), expected).
		Return(nil).
		Times(1)

	svc := service.NewWarehouseService(mockRepo)
	err := svc.ReceiveItem(context.Background(), incoming)
	require.NoError(t, err)
}

// TestReceiveItem_SaveError verifies that a repository error from SaveItem is
// propagated back to the caller.
func TestReceiveItem_SaveError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockWarehouseRepository(ctrl)

	mockRepo.EXPECT().
		GetItemByID(gomock.Any(), "BRG-003").
		Return(domain.Item{}, domain.ErrItemNotFound).
		Times(1)

	mockRepo.EXPECT().
		SaveItem(gomock.Any(), gomock.Any()).
		Return(fmt.Errorf("db connection lost")).
		Times(1)

	svc := service.NewWarehouseService(mockRepo)
	err := svc.ReceiveItem(context.Background(), domain.Item{ID: "BRG-003", Name: "Keyboard", Quantity: 5})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "db connection lost")
}

// ── DispatchItem ─────────────────────────────────────────────────────────────

// TestDispatchItem_Valid verifies that dispatching a valid quantity reduces
// the stock and saves the updated item.
func TestDispatchItem_Valid(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockWarehouseRepository(ctrl)

	existing := domain.Item{ID: "BRG-001", Name: "Laptop", Quantity: 10, Location: "Rak-A1"}

	mockRepo.EXPECT().
		GetItemByID(gomock.Any(), "BRG-001").
		Return(existing, nil).
		Times(1)

	// Quantity must be reduced: 10 - 3 = 7
	expected := domain.Item{ID: "BRG-001", Name: "Laptop", Quantity: 7, Location: "Rak-A1"}
	mockRepo.EXPECT().
		SaveItem(gomock.Any(), expected).
		Return(nil).
		Times(1)

	svc := service.NewWarehouseService(mockRepo)
	err := svc.DispatchItem(context.Background(), "BRG-001", 3)
	require.NoError(t, err)
}

// TestDispatchItem_ExactStock verifies that dispatching exactly the available
// quantity reduces stock to zero.
func TestDispatchItem_ExactStock(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockWarehouseRepository(ctrl)

	existing := domain.Item{ID: "BRG-001", Name: "Laptop", Quantity: 5, Location: "Rak-A1"}

	mockRepo.EXPECT().
		GetItemByID(gomock.Any(), "BRG-001").
		Return(existing, nil).
		Times(1)

	expected := domain.Item{ID: "BRG-001", Name: "Laptop", Quantity: 0, Location: "Rak-A1"}
	mockRepo.EXPECT().
		SaveItem(gomock.Any(), expected).
		Return(nil).
		Times(1)

	svc := service.NewWarehouseService(mockRepo)
	err := svc.DispatchItem(context.Background(), "BRG-001", 5)
	require.NoError(t, err)
}

// TestDispatchItem_OutOfStock verifies that dispatching more than available
// returns ErrOutOfStock and does NOT call SaveItem.
func TestDispatchItem_OutOfStock(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockWarehouseRepository(ctrl)

	existing := domain.Item{ID: "BRG-001", Name: "Laptop", Quantity: 2, Location: "Rak-A1"}

	mockRepo.EXPECT().
		GetItemByID(gomock.Any(), "BRG-001").
		Return(existing, nil).
		Times(1)

	// SaveItem must NOT be called when stock is insufficient
	mockRepo.EXPECT().SaveItem(gomock.Any(), gomock.Any()).Times(0)

	svc := service.NewWarehouseService(mockRepo)
	err := svc.DispatchItem(context.Background(), "BRG-001", 10)
	require.ErrorIs(t, err, domain.ErrOutOfStock)
}

// TestDispatchItem_ItemNotFound verifies that dispatching a non-existent item
// returns ErrItemNotFound and does NOT call SaveItem.
func TestDispatchItem_ItemNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockWarehouseRepository(ctrl)

	mockRepo.EXPECT().
		GetItemByID(gomock.Any(), "BRG-999").
		Return(domain.Item{}, domain.ErrItemNotFound).
		Times(1)

	mockRepo.EXPECT().SaveItem(gomock.Any(), gomock.Any()).Times(0)

	svc := service.NewWarehouseService(mockRepo)
	err := svc.DispatchItem(context.Background(), "BRG-999", 1)
	require.ErrorIs(t, err, domain.ErrItemNotFound)
}

// TestDispatchItem_SaveError verifies that a repository error from SaveItem
// is propagated back to the caller.
func TestDispatchItem_SaveError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockWarehouseRepository(ctrl)

	existing := domain.Item{ID: "BRG-001", Name: "Laptop", Quantity: 10, Location: "Rak-A1"}

	mockRepo.EXPECT().
		GetItemByID(gomock.Any(), "BRG-001").
		Return(existing, nil).
		Times(1)

	mockRepo.EXPECT().
		SaveItem(gomock.Any(), gomock.Any()).
		Return(fmt.Errorf("disk full")).
		Times(1)

	svc := service.NewWarehouseService(mockRepo)
	err := svc.DispatchItem(context.Background(), "BRG-001", 3)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "disk full")
}

// ── CheckStock ───────────────────────────────────────────────────────────────

// TestCheckStock_Found verifies that CheckStock returns the correct item when
// it exists in the repository.
func TestCheckStock_Found(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockWarehouseRepository(ctrl)

	expected := domain.Item{ID: "BRG-001", Name: "Laptop", Quantity: 10, Location: "Rak-A1"}

	mockRepo.EXPECT().
		GetItemByID(gomock.Any(), "BRG-001").
		Return(expected, nil).
		Times(1)

	svc := service.NewWarehouseService(mockRepo)
	item, err := svc.CheckStock(context.Background(), "BRG-001")
	require.NoError(t, err)
	assert.Equal(t, expected, item)
}

// TestCheckStock_NotFound verifies that CheckStock returns ErrItemNotFound
// when the item does not exist.
func TestCheckStock_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockWarehouseRepository(ctrl)

	mockRepo.EXPECT().
		GetItemByID(gomock.Any(), "BRG-999").
		Return(domain.Item{}, domain.ErrItemNotFound).
		Times(1)

	svc := service.NewWarehouseService(mockRepo)
	_, err := svc.CheckStock(context.Background(), "BRG-999")
	require.ErrorIs(t, err, domain.ErrItemNotFound)
}

// TestCheckStock_RepositoryError verifies that an unexpected repository error
// is propagated back to the caller.
func TestCheckStock_RepositoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockWarehouseRepository(ctrl)

	mockRepo.EXPECT().
		GetItemByID(gomock.Any(), "BRG-001").
		Return(domain.Item{}, fmt.Errorf("connection timeout")).
		Times(1)

	svc := service.NewWarehouseService(mockRepo)
	_, err := svc.CheckStock(context.Background(), "BRG-001")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "connection timeout")
}
