package services_test

import (
	"strings"
	"testing"

	"routing-service/internal/services"
)

func TestInterHubKey_Format(t *testing.T) {
	key := services.InterHubKey("JKT", "BDG", "REG")
	if key != "route:JKT:BDG:REG" {
		t.Errorf("unexpected key: %s", key)
	}
}

func TestInterHubKey_StartsWithRoutePrefix(t *testing.T) {
	key := services.InterHubKey("A", "B", "EXP")
	if !strings.HasPrefix(key, "route:") {
		t.Errorf("key %q should start with 'route:'", key)
	}
}

func TestCourierRouteKey_Format(t *testing.T) {
	key := services.CourierRouteKey("courier-123", "2024-01-15")
	if key != "courier_route:courier-123:2024-01-15" {
		t.Errorf("unexpected key: %s", key)
	}
}

func TestCourierRouteKey_StartsWithCourierPrefix(t *testing.T) {
	key := services.CourierRouteKey("c1", "2024-06-01")
	if !strings.HasPrefix(key, "courier_route:") {
		t.Errorf("key %q should start with 'courier_route:'", key)
	}
}

func TestInterHubKey_DifferentServiceTypes(t *testing.T) {
	reg := services.InterHubKey("JKT", "BDG", "REG")
	exp := services.InterHubKey("JKT", "BDG", "EXP")
	if reg == exp {
		t.Error("keys with different service_type should differ")
	}
}
