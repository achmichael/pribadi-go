package scheduler

import (
	"context"
	"fmt"
	"time"

	"github.com/achmichael/pribadi-go/internal/delivery/whatsapp"
	"github.com/achmichael/pribadi-go/internal/domain"
	"github.com/achmichael/pribadi-go/internal/repository"
	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog"
)

// DashboardCronScheduler manages dynamic cron jobs from the dashboard
type DashboardCronScheduler struct {
	cron       *cron.Cron
	repo       repository.DashboardRepository
	waClient   *whatsapp.Client
	logger     *zerolog.Logger
	jobIDs     map[string]cron.EntryID // Maps DB ID to cron EntryID
}

// NewDashboardCronScheduler initializes a dynamic scheduler
func NewDashboardCronScheduler(repo repository.DashboardRepository, waClient *whatsapp.Client, logger *zerolog.Logger) *DashboardCronScheduler {
	return &DashboardCronScheduler{
		cron:       cron.New(),
		repo:       repo,
		waClient:   waClient,
		logger:     logger,
		jobIDs:     make(map[string]cron.EntryID),
	}
}

// Start begins the scheduler and registers initial jobs
func (s *DashboardCronScheduler) Start(ctx context.Context) error {
	s.logger.Info().Msg("Starting dynamic cron scheduler")
	if err := s.ReloadJobs(ctx); err != nil {
		s.logger.Warn().Err(err).Msg("Failed to load initial cron jobs")
	}

	s.cron.Start()

	// Poll every minute to check for new/updated jobs (simple sync strategy)
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				s.cron.Stop()
				return
			case <-ticker.C:
				s.ReloadJobs(ctx)
			}
		}
	}()

	return nil
}

// ReloadJobs synchronizes the running cron jobs with the database
func (s *DashboardCronScheduler) ReloadJobs(ctx context.Context) error {
	jobs, err := s.repo.ListCronJobs(ctx)
	if err != nil {
		return err
	}

	// For simplicity, we remove all existing dynamic jobs and re-add them.
	// In a high-volume system, you'd calculate the diff.
	for _, entryID := range s.jobIDs {
		s.cron.Remove(entryID)
	}
	s.jobIDs = make(map[string]cron.EntryID)

	for _, job := range jobs {
		if !job.IsActive {
			continue
		}

		j := job // capture loop variable
		entryID, err := s.cron.AddFunc(j.ScheduleCron, func() {
			s.executeJob(context.Background(), j)
		})
		if err != nil {
			s.logger.Error().Err(err).Str("job_id", j.ID).Msg("Failed to add cron job")
			continue
		}
		s.jobIDs[j.ID] = entryID
	}

	return nil
}

func (s *DashboardCronScheduler) executeJob(ctx context.Context, job domain.CronJob) {
	s.logger.Info().Str("job_id", job.ID).Str("type", job.JobType).Msg("Executing cron job")

	var output string
	var err error

	// Execute job logic based on type
	switch job.JobType {
	case "stock_alert":
		output, err = s.executeStockAlert(ctx, job)
	case "custom_reminder":
		output, err = s.executeCustomReminder(ctx, job)
	default:
		err = fmt.Errorf("unknown job type: %s", job.JobType)
	}

	status := "success"
	errorDetail := ""
	if err != nil {
		status = "failed"
		errorDetail = err.Error()
		s.logger.Error().Err(err).Str("job_id", job.ID).Msg("Cron job execution failed")
	}

	// Send to WhatsApp if target provided and output exists
	if job.TargetWhatsappJID != "" && output != "" {
		sendErr := s.waClient.SendText(ctx, job.TargetWhatsappJID, output)
		if sendErr != nil {
			s.logger.Error().Err(sendErr).Str("job_id", job.ID).Msg("Failed to send cron output to WhatsApp")
			if status == "success" {
				status = "failed"
				errorDetail = "Failed to send WhatsApp message"
			}
		}
	}

	// Log execution
	logEntry := domain.CronJobLog{
		ID:            uuid.NewString(),
		CronJobID:     job.ID,
		Status:        status,
		OutputMessage: output,
		ErrorDetail:   errorDetail,
	}
	_ = s.repo.LogCronJobExecution(ctx, logEntry)
}

func (s *DashboardCronScheduler) executeStockAlert(ctx context.Context, job domain.CronJob) (string, error) {
	// In a real implementation, you would:
	// 1. Parse job.ConfigJSON to get ticker and condition
	// 2. Fetch latest price
	// 3. Evaluate condition
	// 4. Return message if condition met, else empty string
	return "Stock Alert: Condition checked (mock)", nil
}

func (s *DashboardCronScheduler) executeCustomReminder(ctx context.Context, job domain.CronJob) (string, error) {
	// Parse ConfigJSON for the reminder message
	return "Reminder triggered by Cron!", nil
}
