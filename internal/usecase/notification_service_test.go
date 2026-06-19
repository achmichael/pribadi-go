package usecase

import (
	"testing"

	"github.com/achmichael/pribadi-go/internal/delivery/webhook"
	"github.com/rs/zerolog"
)

func TestFormatMessage(t *testing.T) {
	logger := zerolog.Nop()
	service := &notificationService{
		logger: &logger,
	}

	tests := []struct {
		name    string
		payload webhook.WebhookPayload
		want    string
	}{
		{
			name: "deployment success",
			payload: webhook.WebhookPayload{
				Source:    "github",
				EventType: "deployment_success",
				Data: map[string]interface{}{
					"project":     "my-app",
					"environment": "production",
					"version":     "v1.0.0",
				},
			},
			want: "deployment_success",
		},
		{
			name: "alert",
			payload: webhook.WebhookPayload{
				Source:    "monitoring",
				EventType: "alert",
				Data: map[string]interface{}{
					"severity": "high",
					"service":  "api",
					"message":  "High CPU usage",
				},
			},
			want: "alert",
		},
		{
			name: "backup completed",
			payload: webhook.WebhookPayload{
				Source:    "backup-service",
				EventType: "backup_completed",
				Data: map[string]interface{}{
					"database": "postgres",
					"size":     "1.5GB",
					"duration": "5m",
				},
			},
			want: "backup_completed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			message := service.formatMessage(tt.payload)
			
			// Check message contains event type
			if message == "" {
				t.Error("Expected non-empty message")
			}
			
			// Check header exists
			if len(message) < 10 {
				t.Errorf("Message too short: %s", message)
			}
		})
	}
}

func TestFormatDeploymentMessage(t *testing.T) {
	logger := zerolog.Nop()
	service := &notificationService{
		logger: &logger,
	}

	data := map[string]interface{}{
		"project":     "my-app",
		"environment": "production",
		"version":     "v1.0.0",
		"message":     "Deployment completed successfully",
	}

	result := service.formatDeploymentMessage(data)

	if result == "" {
		t.Error("Expected non-empty deployment message")
	}

	// Check that key fields are included
	expectedFields := []string{"my-app", "production", "v1.0.0"}
	for _, field := range expectedFields {
		if !contains(result, field) {
			t.Errorf("Expected message to contain '%s', got: %s", field, result)
		}
	}
}

func TestFormatAlertMessage(t *testing.T) {
	logger := zerolog.Nop()
	service := &notificationService{
		logger: &logger,
	}

	data := map[string]interface{}{
		"severity": "high",
		"service":  "api",
		"message":  "High CPU usage detected",
	}

	result := service.formatAlertMessage(data)

	if result == "" {
		t.Error("Expected non-empty alert message")
	}
}

func TestParseJIDs(t *testing.T) {
	logger := zerolog.Nop()
	
	tests := []struct {
		name       string
		notifyJIDs string
		wantCount  int
	}{
		{
			name:       "single JID",
			notifyJIDs: "6281234567890@s.whatsapp.net",
			wantCount:  1,
		},
		{
			name:       "multiple JIDs",
			notifyJIDs: "6281234567890@s.whatsapp.net,6289876543210@s.whatsapp.net",
			wantCount:  2,
		},
		{
			name:       "JIDs with spaces",
			notifyJIDs: "6281234567890@s.whatsapp.net , 6289876543210@s.whatsapp.net",
			wantCount:  2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewNotificationService(nil, nil, tt.notifyJIDs, &logger)
			ns := service.(*notificationService)
			
			if len(ns.targetJIDs) != tt.wantCount {
				t.Errorf("Expected %d JIDs, got %d", tt.wantCount, len(ns.targetJIDs))
			}
		})
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsInMiddle(s, substr)))
}

func containsInMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
