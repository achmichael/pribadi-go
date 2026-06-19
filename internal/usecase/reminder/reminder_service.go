package reminder

import (
	"context"
	"time"

	"github.com/achmichael/pribadi-go/internal/repository"
	"github.com/achmichael/pribadi-go/internal/repository/sqlc"
)

type ReminderService interface {
	CreateReminder(ctx context.Context, jid, message string, scheduledAt time.Time) error
}

type reminderService struct {
	repo repository.Repository
}

func NewReminderService(repo repository.Repository) ReminderService {
	return &reminderService{repo}
}

func (s *reminderService) CreateReminder(ctx context.Context, jid, message string, scheduledAt time.Time) error {
	_, err := s.repo.InsertReminder(ctx, sqlc.InsertReminderParams{
		Jid:         jid,
		Message:     message,
		ScheduledAt: scheduledAt,
	})
	return err
}
