package response

import (
	"context"
	"encoding/json"
	"time"

	"github.com/achmichael/pribadi-go/internal/repository"
	"github.com/rs/zerolog"
)

// ─── Types ─────────────────────────────────────────────────────────

// InteractionLog captures a complete turn for analytics and debugging.
type InteractionLog struct {
	UserID       string                 `json:"user_id"`
	SessionID    string                 `json:"session_id"`
	Timestamp    time.Time              `json:"timestamp"`
	UserMessage  string                 `json:"user_message"`
	Response     string                 `json:"response"`
	Metadata     map[string]interface{} `json:"metadata"`
	Duration     int64                  `json:"duration_ms"`
	Success      bool                   `json:"success"`
	ErrorMessage string                 `json:"error_message,omitempty"`
}

// ─── Interface ─────────────────────────────────────────────────────

// InteractionLogger logs complete conversation turns.
type InteractionLogger interface {
	// Log records an interaction with full metadata.
	Log(ctx context.Context, log InteractionLog) error
	
	// LogError records a failed interaction.
	LogError(ctx context.Context, userID, sessionID, userMessage, errorMsg string, duration int64) error
}

// ─── Implementation ────────────────────────────────────────────────

type interactionLogger struct {
	repo   repository.Repository
	logger *zerolog.Logger
}

// NewInteractionLogger creates an InteractionLogger.
func NewInteractionLogger(repo repository.Repository, logger *zerolog.Logger) InteractionLogger {
	return &interactionLogger{
		repo:   repo,
		logger: logger,
	}
}

func (l *interactionLogger) Log(ctx context.Context, log InteractionLog) error {
	// Convert metadata to JSON
	metadataJSON := "{}"
	if len(log.Metadata) > 0 {
		if bytes, err := json.Marshal(log.Metadata); err == nil {
			metadataJSON = string(bytes)
		}
	}
	
	// Store in repository (if it has interaction logging support)
	// For now, just log to zerolog
	l.logger.Info().
		Str("user_id", log.UserID).
		Str("session_id", log.SessionID).
		Int64("duration_ms", log.Duration).
		Bool("success", log.Success).
		Str("metadata", metadataJSON).
		Msg("[interaction] logged")
	
	return nil
}

func (l *interactionLogger) LogError(ctx context.Context, userID, sessionID, userMessage, errorMsg string, duration int64) error {
	l.logger.Error().
		Str("user_id", userID).
		Str("session_id", sessionID).
		Str("user_message", userMessage).
		Str("error", errorMsg).
		Int64("duration_ms", duration).
		Msg("[interaction] error logged")
	
	return nil
}

// ─── Metrics Aggregator ────────────────────────────────────────────

// MetricsAggregator computes statistics from response metadata.
type MetricsAggregator interface {
	// Aggregate computes metrics from a batch of interactions.
	Aggregate(logs []InteractionLog) Metrics
}

// Metrics holds computed statistics.
type Metrics struct {
	TotalInteractions int
	SuccessRate       float32
	AvgDurationMs     float32
	AvgConfidence     float32
	HighRiskCount     int
	
	// By intent
	IntentBreakdown map[string]int
	
	// By response strategy
	StrategyBreakdown map[string]int
}

type metricsAggregator struct{}

// NewMetricsAggregator creates a MetricsAggregator.
func NewMetricsAggregator() MetricsAggregator {
	return &metricsAggregator{}
}

func (m *metricsAggregator) Aggregate(logs []InteractionLog) Metrics {
	if len(logs) == 0 {
		return Metrics{}
	}
	
	metrics := Metrics{
		TotalInteractions: len(logs),
		IntentBreakdown:   make(map[string]int),
		StrategyBreakdown: make(map[string]int),
	}
	
	var totalDuration int64
	var totalConfidence float32
	successCount := 0
	
	for _, log := range logs {
		if log.Success {
			successCount++
		}
		totalDuration += log.Duration
		
		// Extract confidence from metadata
		if conf, ok := log.Metadata["confidence_score"].(float64); ok {
			totalConfidence += float32(conf)
		}
		
		// Extract hallucination risk
		if risk, ok := log.Metadata["hallucination_risk"].(string); ok {
			if risk == "high" {
				metrics.HighRiskCount++
			}
		}
		
		// Extract intent
		if intent, ok := log.Metadata["intent"].(string); ok {
			metrics.IntentBreakdown[intent]++
		}
		
		// Extract strategy
		if strategy, ok := log.Metadata["strategy"].(string); ok {
			metrics.StrategyBreakdown[strategy]++
		}
	}
	
	metrics.SuccessRate = float32(successCount) / float32(len(logs))
	metrics.AvgDurationMs = float32(totalDuration) / float32(len(logs))
	
	if successCount > 0 {
		metrics.AvgConfidence = totalConfidence / float32(successCount)
	}
	
	return metrics
}
