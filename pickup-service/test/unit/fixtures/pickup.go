package fixtures

import (
	"time"

	"github.com/pickup-service/internal/domain"
)

// ValidPickup returns a fully populated Pickup with sensible defaults.
func ValidPickup() *domain.Pickup {
	now := time.Now()
	return &domain.Pickup{
		PickupID:       "pickup-123",
		OrderID:        "order-456",
		UserID:         "user-789",
		Status:         domain.StatusScheduled,
		PickupAddress:  "123 Main St",
		PickupCityCode: "JKT",
		ContactName:    "John Doe",
		ContactPhone:   "+62812345678",
		AttemptCount:   0,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// PickupWithStatus returns a ValidPickup with the given status applied.
func PickupWithStatus(s domain.Status) *domain.Pickup {
	p := ValidPickup()
	p.Status = s
	return p
}
