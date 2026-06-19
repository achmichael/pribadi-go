package webhook

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/rs/zerolog"
)

// WebhookPayload represents the incoming webhook payload
type WebhookPayload struct {
	Source    string                 `json:"source"`
	EventType string                 `json:"event_type"`
	Data      map[string]interface{} `json:"data"`
}

// NotificationService defines the interface for handling notifications
type NotificationService interface {
	Handle(ctx context.Context, payload WebhookPayload) error
}

// Handler handles webhook requests
type Handler struct {
	notificationService NotificationService
	webhookSecret       string
	logger              *zerolog.Logger
}

// NewHandler creates a new webhook handler
func NewHandler(notificationService NotificationService, webhookSecret string, logger *zerolog.Logger) *Handler {
	return &Handler{
		notificationService: notificationService,
		webhookSecret:       webhookSecret,
		logger:              logger,
	}
}

// HandleWebhook handles POST /webhook requests
func (h *Handler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	// Read body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to read request body")
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Validate HMAC signature
	signature := r.Header.Get("X-Webhook-Signature")
	if !h.validateSignature(body, signature) {
		h.logger.Warn().Msg("Invalid webhook signature")
		http.Error(w, "Invalid signature", http.StatusUnauthorized)
		return
	}

	// Parse payload
	var payload WebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		h.logger.Error().Err(err).Msg("Failed to parse webhook payload")
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	h.logger.Info().
		Str("source", payload.Source).
		Str("event_type", payload.EventType).
		Msg("Webhook received")

	// Return 200 immediately
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"accepted"}`))

	// Process asynchronously
	go func() {
		ctx := context.Background()
		if err := h.notificationService.Handle(ctx, payload); err != nil {
			h.logger.Error().
				Err(err).
				Str("source", payload.Source).
				Str("event_type", payload.EventType).
				Msg("Failed to process webhook")
		}
	}()
}

// validateSignature validates HMAC-SHA256 signature
func (h *Handler) validateSignature(body []byte, signature string) bool {
	if h.webhookSecret == "" {
		// No secret configured, skip validation
		return true
	}

	if signature == "" {
		return false
	}

	// Compute expected signature
	mac := hmac.New(sha256.New, []byte(h.webhookSecret))
	mac.Write(body)
	expectedSignature := hex.EncodeToString(mac.Sum(nil))

	// Compare signatures
	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

// ValidatePayload validates the webhook payload structure
func ValidatePayload(payload WebhookPayload) error {
	if payload.Source == "" {
		return fmt.Errorf("source is required")
	}
	if payload.EventType == "" {
		return fmt.Errorf("event_type is required")
	}
	if payload.Data == nil {
		return fmt.Errorf("data is required")
	}
	return nil
}
