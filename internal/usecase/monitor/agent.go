package monitor

import (
	"context"
	"time"

	"github.com/achmichael/pribadi-go/internal/domain"
	"github.com/rs/zerolog"
)

type DeliveryService interface {
	Send(ctx context.Context, platform string, userID string, message string) error
}

type Agent struct {
	taskRepo     domain.MonitorTaskRepository
	historyRepo  domain.MonitorValueHistoryRepository
	providers    map[domain.SourceType]domain.DataSourceProvider
	evaluators   map[domain.ConditionMode]domain.ConditionEvaluator
	notifier     *Notifier
	delivery     DeliveryService
	logger       *zerolog.Logger
	baseInterval time.Duration
}

func NewAgent(
	taskRepo domain.MonitorTaskRepository,
	historyRepo domain.MonitorValueHistoryRepository,
	providers map[domain.SourceType]domain.DataSourceProvider,
	evaluators map[domain.ConditionMode]domain.ConditionEvaluator,
	notifier *Notifier,
	delivery DeliveryService,
	logger *zerolog.Logger,
) *Agent {
	return &Agent{
		taskRepo:     taskRepo,
		historyRepo:  historyRepo,
		providers:    providers,
		evaluators:   evaluators,
		notifier:     notifier,
		delivery:     delivery,
		logger:       logger,
		baseInterval: 30 * time.Second,
	}
}

func (a *Agent) Run(ctx context.Context) error {
	ticker := time.NewTicker(a.baseInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := a.cycle(ctx); err != nil {
				a.logger.Error().Err(err).Msg("monitor cycle failed")
			}
		}
	}
}

func (a *Agent) cycle(ctx context.Context) error {
	tasks, err := a.taskRepo.ListActive(ctx)
	if err != nil {
		return err
	}
	for _, task := range tasks {
		if !a.isDue(task) {
			continue
		}
		a.processTask(ctx, task)
	}
	return nil
}

func (a *Agent) processTask(ctx context.Context, task domain.MonitorTask) {
	provider, ok := a.providers[task.SourceType]
	if !ok {
		a.logger.Warn().Int64("task_id", task.ID).Str("source_type", string(task.SourceType)).Msg("unknown source type")
		return
	}

	numeric, text, rawSnapshot, err := provider.Fetch(ctx, task)
	if err != nil {
		a.logger.Warn().Int64("task_id", task.ID).Err(err).Msg("fetch failed")
		return
	}

	evaluator, ok := a.evaluators[task.ConditionMode]
	if !ok {
		a.logger.Warn().Int64("task_id", task.ID).Str("mode", string(task.ConditionMode)).Msg("unknown condition mode")
		return
	}

	met, err := evaluator.Evaluate(ctx, task, numeric, text, task.LastValue, rawSnapshot)
	if err != nil {
		a.logger.Warn().Int64("task_id", task.ID).Err(err).Msg("evaluation failed")
		return
	}

	if met {
		msg, err := a.notifier.GenerateMessage(ctx, task, numeric, text)
		if err == nil {
			_ = a.delivery.Send(ctx, task.Platform, task.UserID, msg)
			_ = a.taskRepo.UpdateLastNotified(ctx, task.ID, time.Now())
		}
	}

	_ = a.taskRepo.UpdateLastValue(ctx, task.ID, numeric, text, time.Now())

	snapshotHash := hashSnapshot(rawSnapshot)
	_ = a.historyRepo.Insert(ctx, task.ID, numeric, text, snapshotHash)
}

func (a *Agent) isDue(task domain.MonitorTask) bool {
	if task.LastCheckedAt == nil {
		return true
	}
	return time.Since(*task.LastCheckedAt) >= time.Duration(task.PollIntervalSeconds)*time.Second
}
