package scheduler

import (
	"context"
	"time"

	"github.com/achmichael/pribadi-go/internal/delivery/whatsapp"
	"github.com/achmichael/pribadi-go/internal/repository"
	"github.com/go-co-op/gocron/v2"
	"github.com/rs/zerolog"
)

type ReminderScheduler struct {
	waClient   *whatsapp.Client
	repo       repository.Repository
	logger     *zerolog.Logger
	scheduler  gocron.Scheduler
}

func NewReminderScheduler(waClient *whatsapp.Client, repo repository.Repository, logger *zerolog.Logger) (*ReminderScheduler, error) {
	s, err := gocron.NewScheduler()
	if err != nil {
		return nil, err
	}
	return &ReminderScheduler{waClient, repo, logger, s}, nil
}

func (s *ReminderScheduler) Start(ctx context.Context) error {
	_, err := s.scheduler.NewJob(
		gocron.DurationJob(1*time.Minute),
		gocron.NewTask(s.checkReminders, ctx),
	)
	if err != nil {
		return err
	}
	s.scheduler.Start()
	return nil
}

func (s *ReminderScheduler) checkReminders(ctx context.Context) {
	reminders, err := s.repo.ListPendingReminders(ctx, time.Now())
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to list reminders")
		return
	}
	for _, r := range reminders {
		err := s.waClient.SendText(ctx, r.Jid, r.Message)
		if err != nil {
			s.logger.Error().Err(err).Msg("Failed to send reminder")
			continue
		}
		s.repo.MarkReminderAsSent(ctx, r.ID)
		s.logger.Info().Int64("id", r.ID).Msg("Reminder sent")
	}
}
