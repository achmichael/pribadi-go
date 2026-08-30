package scheduler

import (
	"context"

	"github.com/achmichael/pribadi-go/internal/usecase/monitor"
	"github.com/rs/zerolog"
)

type MonitorScheduler struct {
	agent  *monitor.Agent
	logger *zerolog.Logger
}

func NewMonitorScheduler(agent *monitor.Agent, logger *zerolog.Logger) *MonitorScheduler {
	return &MonitorScheduler{agent: agent, logger: logger}
}

func (s *MonitorScheduler) Start(ctx context.Context) error {
	s.logger.Info().Msg("Starting monitor scheduler")
	return s.agent.Run(ctx)
}
