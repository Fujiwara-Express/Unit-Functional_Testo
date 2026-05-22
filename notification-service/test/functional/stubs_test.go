package functional_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
)

// FirebaseCall records a single call to the Firebase stub.
type FirebaseCall struct {
	UserID  string
	Message string
}

type firebaseStubServer struct {
	server     *httptest.Server
	mu         sync.Mutex
	calls      []FirebaseCall
	statusCode int
}

func newFirebaseStub() *firebaseStubServer {
	s := &firebaseStubServer{statusCode: http.StatusOK}
	s.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/fcm/send" && r.Method == http.MethodPost {
			var req struct {
				UserID  string `json:"user_id"`
				Message string `json:"message"`
			}
			json.NewDecoder(r.Body).Decode(&req)
			s.mu.Lock()
			s.calls = append(s.calls, FirebaseCall{req.UserID, req.Message})
			code := s.statusCode
			s.mu.Unlock()
			w.WriteHeader(code)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	return s
}

func (s *firebaseStubServer) URL() string { return s.server.URL }
func (s *firebaseStubServer) Close()      { s.server.Close() }
func (s *firebaseStubServer) Reset() {
	s.mu.Lock()
	s.calls = nil
	s.statusCode = http.StatusOK
	s.mu.Unlock()
}
func (s *firebaseStubServer) SetStatus(code int) {
	s.mu.Lock()
	s.statusCode = code
	s.mu.Unlock()
}
func (s *firebaseStubServer) Calls() []FirebaseCall {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]FirebaseCall(nil), s.calls...)
}

// SendGridCall records a single call to the SendGrid stub.
type SendGridCall struct {
	Recipient string
	Subject   string
	Body      string
}

type sendgridStubServer struct {
	server     *httptest.Server
	mu         sync.Mutex
	calls      []SendGridCall
	statusCode int
}

func newSendGridStub() *sendgridStubServer {
	s := &sendgridStubServer{statusCode: http.StatusOK}
	s.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v3/mail/send" && r.Method == http.MethodPost {
			var req struct {
				Recipient string `json:"recipient"`
				Subject   string `json:"subject"`
				Body      string `json:"body"`
			}
			json.NewDecoder(r.Body).Decode(&req)
			s.mu.Lock()
			s.calls = append(s.calls, SendGridCall{req.Recipient, req.Subject, req.Body})
			code := s.statusCode
			s.mu.Unlock()
			w.WriteHeader(code)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	return s
}

func (s *sendgridStubServer) URL() string { return s.server.URL }
func (s *sendgridStubServer) Close()      { s.server.Close() }
func (s *sendgridStubServer) Reset() {
	s.mu.Lock()
	s.calls = nil
	s.statusCode = http.StatusOK
	s.mu.Unlock()
}
func (s *sendgridStubServer) SetStatus(code int) {
	s.mu.Lock()
	s.statusCode = code
	s.mu.Unlock()
}
func (s *sendgridStubServer) Calls() []SendGridCall {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]SendGridCall(nil), s.calls...)
}

// WhatsAppCall records a single call to the WhatsApp stub.
type WhatsAppCall struct {
	Phone   string
	Message string
}

type whatsappStubServer struct {
	server     *httptest.Server
	mu         sync.Mutex
	calls      []WhatsAppCall
	statusCode int
}

func newWhatsAppStub() *whatsappStubServer {
	s := &whatsappStubServer{statusCode: http.StatusOK}
	s.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/messages" && r.Method == http.MethodPost {
			var req struct {
				Phone   string `json:"phone"`
				Message string `json:"message"`
			}
			json.NewDecoder(r.Body).Decode(&req)
			s.mu.Lock()
			s.calls = append(s.calls, WhatsAppCall{req.Phone, req.Message})
			code := s.statusCode
			s.mu.Unlock()
			w.WriteHeader(code)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	return s
}

func (s *whatsappStubServer) URL() string { return s.server.URL }
func (s *whatsappStubServer) Close()      { s.server.Close() }
func (s *whatsappStubServer) Reset() {
	s.mu.Lock()
	s.calls = nil
	s.statusCode = http.StatusOK
	s.mu.Unlock()
}
func (s *whatsappStubServer) SetStatus(code int) {
	s.mu.Lock()
	s.statusCode = code
	s.mu.Unlock()
}
func (s *whatsappStubServer) Calls() []WhatsAppCall {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]WhatsAppCall(nil), s.calls...)
}
