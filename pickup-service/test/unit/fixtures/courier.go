package fixtures

import (
	"encoding/json"

	"github.com/pickup-service/internal/client"
)

// ValidCourier returns a Courier with sensible defaults.
func ValidCourier() client.Courier {
	return client.Courier{
		CourierID: "courier-001",
		Name:      "Jane Smith",
		CityCode:  "JKT",
	}
}

// ValidCourierJSON returns the JSON encoding of ValidCourier.
func ValidCourierJSON() []byte {
	b, _ := json.Marshal(ValidCourier())
	return b
}
