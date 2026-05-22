package domain

import "fmt"

// Validate checks that all required fields are present in the Pickup.
// It returns an error naming the first missing field found.
func (p *Pickup) Validate() error {
	if p.OrderID == "" {
		return fmt.Errorf("%w: missing required field order_id", ErrValidation)
	}
	if p.UserID == "" {
		return fmt.Errorf("%w: missing required field user_id", ErrValidation)
	}
	if p.PickupAddress == "" {
		return fmt.Errorf("%w: missing required field pickup_address", ErrValidation)
	}
	if p.PickupCityCode == "" {
		return fmt.Errorf("%w: missing required field pickup_city_code", ErrValidation)
	}
	if p.ContactName == "" {
		return fmt.Errorf("%w: missing required field contact_name", ErrValidation)
	}
	if p.ContactPhone == "" {
		return fmt.Errorf("%w: missing required field contact_phone", ErrValidation)
	}
	return nil
}
