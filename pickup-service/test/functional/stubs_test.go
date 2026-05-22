package functional_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"

	"github.com/pickup-service/internal/client"
)

// NotifyCall records a single call to the notification stub.
type NotifyCall struct {
	ContactName  string
	ContactPhone string
	CourierID    string
}

type notificationStubServer struct {
	server *httptest.Server
	mu     sync.Mutex
	calls  []NotifyCall
}

func newNotificationStub() *notificationStubServer {
	s := &notificationStubServer{}
	s.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/notifications/courier-en-route" && r.Method == http.MethodPost {
			var req struct {
				ContactName  string `json:"contact_name"`
				ContactPhone string `json:"contact_phone"`
				CourierID    string `json:"courier_id"`
			}
			_ = json.NewDecoder(r.Body).Decode(&req)
			s.mu.Lock()
			s.calls = append(s.calls, NotifyCall{
				ContactName:  req.ContactName,
				ContactPhone: req.ContactPhone,
				CourierID:    req.CourierID,
			})
			s.mu.Unlock()
		}
		w.WriteHeader(http.StatusOK)
	}))
	return s
}

func (s *notificationStubServer) URL() string { return s.server.URL }
func (s *notificationStubServer) Close()      { s.server.Close() }
func (s *notificationStubServer) Reset() {
	s.mu.Lock()
	s.calls = nil
	s.mu.Unlock()
}
func (s *notificationStubServer) Calls() []NotifyCall {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]NotifyCall(nil), s.calls...)
}

// DeliveryCall records a single call to the delivery stub.
type DeliveryCall struct {
	CityCode string
}

type deliveryStubServer struct {
	server   *httptest.Server
	mu       sync.Mutex
	calls    []DeliveryCall
	couriers []client.Courier
}

func newDeliveryStub() *deliveryStubServer {
	s := &deliveryStubServer{}
	s.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/couriers") && r.Method == http.MethodGet {
			s.mu.Lock()
			s.calls = append(s.calls, DeliveryCall{CityCode: r.URL.Query().Get("city_code")})
			couriers := s.couriers
			s.mu.Unlock()
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(couriers)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	return s
}

func (s *deliveryStubServer) URL() string { return s.server.URL }
func (s *deliveryStubServer) Close()      { s.server.Close() }
func (s *deliveryStubServer) Reset() {
	s.mu.Lock()
	s.calls = nil
	s.mu.Unlock()
}
func (s *deliveryStubServer) SetCouriers(c []client.Courier) {
	s.mu.Lock()
	s.couriers = c
	s.mu.Unlock()
}
func (s *deliveryStubServer) Calls() []DeliveryCall {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]DeliveryCall(nil), s.calls...)
}
