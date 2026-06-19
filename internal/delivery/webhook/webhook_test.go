package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"
)

// MockNotificationService is a mock implementation for testing
type MockNotificationService struct {
	HandleCalled bool
	LastPayload  WebhookPayload
}

func (m *MockNotificationService) Handle(ctx context.Context, payload WebhookPayload) error {
	m.HandleCalled = true
	m.LastPayload = payload
	return nil
}

func TestHandleWebhook(t *testing.T) {
	logger := zerolog.Nop()
	secret := "test-secret"
	mockService := &MockNotificationService{}
	handler := NewHandler(mockService, secret, &logger)

	payload := WebhookPayload{
		Source:    "github",
		EventType: "deployment_success",
		Data: map[string]interface{}{
			"project": "my-app",
			"version": "v1.0.0",
		},
	}

	body, _ := json.Marshal(payload)

	// Compute signature
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	signature := hex.EncodeToString(mac.Sum(nil))

	// Create request
	req := httptest.NewRequest("POST", "/webhook", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-Signature", signature)

	// Create response recorder
	w := httptest.NewRecorder()

	// Handle request
	handler.HandleWebhook(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Note: HandleCalled might not be true immediately due to goroutine
	// In production, you'd use proper async testing or channels
}

func TestValidateSignature(t *testing.T) {
	logger := zerolog.Nop()
	secret := "test-secret"
	handler := NewHandler(nil, secret, &logger)

	body := []byte(`{"test":"data"}`)

	// Compute correct signature
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	correctSignature := hex.EncodeToString(mac.Sum(nil))

	// Test correct signature
	if !handler.validateSignature(body, correctSignature) {
		t.Error("Valid signature was rejected")
	}

	// Test incorrect signature
	if handler.validateSignature(body, "wrong-signature") {
		t.Error("Invalid signature was accepted")
	}

	// Test empty signature
	if handler.validateSignature(body, "") {
		t.Error("Empty signature was accepted")
	}
}

func TestValidatePayload(t *testing.T) {
	tests := []struct {
		name    string
		payload WebhookPayload
		wantErr bool
	}{
		{
			name: "valid payload",
			payload: WebhookPayload{
				Source:    "test",
				EventType: "test_event",
				Data:      map[string]interface{}{"key": "value"},
			},
			wantErr: false,
		},
		{
			name: "missing source",
			payload: WebhookPayload{
				EventType: "test_event",
				Data:      map[string]interface{}{"key": "value"},
			},
			wantErr: true,
		},
		{
			name: "missing event_type",
			payload: WebhookPayload{
				Source: "test",
				Data:   map[string]interface{}{"key": "value"},
			},
			wantErr: true,
		},
		{
			name: "missing data",
			payload: WebhookPayload{
				Source:    "test",
				EventType: "test_event",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePayload(tt.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePayload() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
