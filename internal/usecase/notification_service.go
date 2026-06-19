package usecase

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/achmichael/pribadi-go/internal/delivery/webhook"
	"github.com/achmichael/pribadi-go/internal/delivery/whatsapp"
	"github.com/achmichael/pribadi-go/internal/repository"
	"github.com/achmichael/pribadi-go/internal/repository/sqlc"
	"github.com/rs/zerolog"
)

// NotificationService handles webhook notifications
type NotificationService interface {
	Handle(ctx context.Context, payload webhook.WebhookPayload) error
}

type notificationService struct {
	waClient   *whatsapp.Client
	repo       repository.Repository
	targetJIDs []string
	logger     *zerolog.Logger
}

// NewNotificationService creates a new notification service
func NewNotificationService(
	waClient *whatsapp.Client,
	repo repository.Repository,
	notifyJIDs string,
	logger *zerolog.Logger,
) NotificationService {
	// Parse comma-separated JIDs
	jids := strings.Split(notifyJIDs, ",")
	for i, jid := range jids {
		jids[i] = strings.TrimSpace(jid)
	}

	return &notificationService{
		waClient:   waClient,
		repo:       repo,
		targetJIDs: jids,
		logger:     logger,
	}
}

// Handle processes webhook payload and sends notifications
func (s *notificationService) Handle(ctx context.Context, payload webhook.WebhookPayload) error {
	// Format message based on event type
	message := s.formatMessage(payload)

	s.logger.Info().
		Str("event_type", payload.EventType).
		Int("recipients", len(s.targetJIDs)).
		Msg("Processing notification")

	// Send to all target JIDs
	for _, jid := range s.targetJIDs {
		if jid == "" {
			continue
		}

		// Send message
		err := s.waClient.SendText(ctx, jid, message)
		if err != nil {
			s.logger.Error().
				Err(err).
				Str("jid", jid).
				Msg("Failed to send notification")
			continue
		}

		// Save to database
		_, err = s.repo.InsertMessage(ctx, sqlc.InsertMessageParams{
			WaID:      fmt.Sprintf("notif-%d", time.Now().UnixNano()),
			FromJid:   "system@webhook",
			ToJid:     jid,
			Content:   message,
			MediaType: sql.NullString{},
			Timestamp: time.Now().Unix(),
		})
		if err != nil {
			s.logger.Error().
				Err(err).
				Str("jid", jid).
				Msg("Failed to save notification to database")
		}

		s.logger.Info().
			Str("jid", jid).
			Str("event_type", payload.EventType).
			Msg("Notification sent")
	}

	return nil
}

// formatMessage formats webhook payload into human-readable message
func (s *notificationService) formatMessage(payload webhook.WebhookPayload) string {
	var sb strings.Builder

	// Header
	sb.WriteString("🔔 *Notification*\n\n")

	// Format based on event type
	switch payload.EventType {
	case "deployment_success":
		sb.WriteString("✅ *Deployment Successful*\n")
		sb.WriteString(s.formatDeploymentMessage(payload.Data))

	case "deployment_failed":
		sb.WriteString("❌ *Deployment Failed*\n")
		sb.WriteString(s.formatDeploymentMessage(payload.Data))

	case "alert":
		sb.WriteString("⚠️ *Alert*\n")
		sb.WriteString(s.formatAlertMessage(payload.Data))

	case "backup_completed":
		sb.WriteString("💾 *Backup Completed*\n")
		sb.WriteString(s.formatBackupMessage(payload.Data))

	case "task_reminder":
		sb.WriteString("📋 *Task Reminder*\n")
		sb.WriteString(s.formatTaskMessage(payload.Data))

	default:
		sb.WriteString(fmt.Sprintf("📬 *%s*\n", payload.EventType))
		sb.WriteString(s.formatGenericMessage(payload.Data))
	}

	// Footer
	sb.WriteString(fmt.Sprintf("\n_Source: %s_\n", payload.Source))
	sb.WriteString(fmt.Sprintf("_Time: %s_", time.Now().Format("2006-01-02 15:04:05")))

	return sb.String()
}

// formatDeploymentMessage formats deployment-related messages
func (s *notificationService) formatDeploymentMessage(data map[string]interface{}) string {
	var sb strings.Builder

	if project, ok := data["project"].(string); ok {
		sb.WriteString(fmt.Sprintf("Project: %s\n", project))
	}
	if env, ok := data["environment"].(string); ok {
		sb.WriteString(fmt.Sprintf("Environment: %s\n", env))
	}
	if version, ok := data["version"].(string); ok {
		sb.WriteString(fmt.Sprintf("Version: %s\n", version))
	}
	if message, ok := data["message"].(string); ok {
		sb.WriteString(fmt.Sprintf("Message: %s\n", message))
	}

	return sb.String()
}

// formatAlertMessage formats alert messages
func (s *notificationService) formatAlertMessage(data map[string]interface{}) string {
	var sb strings.Builder

	if severity, ok := data["severity"].(string); ok {
		sb.WriteString(fmt.Sprintf("Severity: %s\n", severity))
	}
	if service, ok := data["service"].(string); ok {
		sb.WriteString(fmt.Sprintf("Service: %s\n", service))
	}
	if message, ok := data["message"].(string); ok {
		sb.WriteString(fmt.Sprintf("Message: %s\n", message))
	}
	if details, ok := data["details"].(string); ok {
		sb.WriteString(fmt.Sprintf("Details: %s\n", details))
	}

	return sb.String()
}

// formatBackupMessage formats backup messages
func (s *notificationService) formatBackupMessage(data map[string]interface{}) string {
	var sb strings.Builder

	if database, ok := data["database"].(string); ok {
		sb.WriteString(fmt.Sprintf("Database: %s\n", database))
	}
	if size, ok := data["size"].(string); ok {
		sb.WriteString(fmt.Sprintf("Size: %s\n", size))
	}
	if location, ok := data["location"].(string); ok {
		sb.WriteString(fmt.Sprintf("Location: %s\n", location))
	}
	if duration, ok := data["duration"].(string); ok {
		sb.WriteString(fmt.Sprintf("Duration: %s\n", duration))
	}

	return sb.String()
}

// formatTaskMessage formats task reminder messages
func (s *notificationService) formatTaskMessage(data map[string]interface{}) string {
	var sb strings.Builder

	if title, ok := data["title"].(string); ok {
		sb.WriteString(fmt.Sprintf("Task: %s\n", title))
	}
	if dueDate, ok := data["due_date"].(string); ok {
		sb.WriteString(fmt.Sprintf("Due: %s\n", dueDate))
	}
	if priority, ok := data["priority"].(string); ok {
		sb.WriteString(fmt.Sprintf("Priority: %s\n", priority))
	}
	if description, ok := data["description"].(string); ok {
		sb.WriteString(fmt.Sprintf("Description: %s\n", description))
	}

	return sb.String()
}

// formatGenericMessage formats any other message type
func (s *notificationService) formatGenericMessage(data map[string]interface{}) string {
	var sb strings.Builder

	for key, value := range data {
		sb.WriteString(fmt.Sprintf("%s: %v\n", key, value))
	}

	return sb.String()
}
