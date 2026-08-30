package monitor

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/achmichael/pribadi-go/internal/domain"
	"github.com/achmichael/pribadi-go/pkg/llm"
	"github.com/rs/zerolog"
)

type NLJudgeEvaluator struct {
	llmClient   llm.Client
	budgetRepo  domain.NLJudgeBudgetRepository
	historyRepo domain.MonitorValueHistoryRepository
	taskRepo    domain.MonitorTaskRepository
	logger      *zerolog.Logger
}

func NewNLJudgeEvaluator(
	llmClient llm.Client,
	budgetRepo domain.NLJudgeBudgetRepository,
	historyRepo domain.MonitorValueHistoryRepository,
	taskRepo domain.MonitorTaskRepository,
	logger *zerolog.Logger,
) *NLJudgeEvaluator {
	return &NLJudgeEvaluator{
		llmClient:   llmClient,
		budgetRepo:  budgetRepo,
		historyRepo: historyRepo,
		taskRepo:    taskRepo,
		logger:      logger,
	}
}

type nlJudgeResponse struct {
	ConditionMet bool   `json:"condition_met"`
	Reasoning    string `json:"reasoning"`
}

func (e *NLJudgeEvaluator) Evaluate(ctx context.Context, task domain.MonitorTask, currentNumeric *float64, currentText string, previousNumeric *float64, rawSnapshot string) (bool, error) {
	windowStart := time.Now().Truncate(24 * time.Hour)
	callCount, err := e.budgetRepo.GetCallCount(ctx, task.ID, windowStart)
	if err != nil {
		return false, fmt.Errorf("get call count: %w", err)
	}

	dailyLimit, err := e.budgetRepo.GetDailyLimit(ctx, task.ID)
	if err != nil {
		return false, fmt.Errorf("get daily limit: %w", err)
	}

	if callCount >= dailyLimit {
		_ = e.taskRepo.MarkStatus(ctx, task.ID, "budget_exceeded")
		e.logger.Warn().Int64("task_id", task.ID).Int("calls", callCount).Msg("nl_judge budget exceeded")
		return false, nil
	}

	currentHash := hashSnapshot(rawSnapshot)
	lastHash, err := e.historyRepo.GetLastSnapshotHash(ctx, task.ID)
	if err != nil {
		return false, fmt.Errorf("get last hash: %w", err)
	}
	if currentHash == lastHash && lastHash != "" {
		return false, nil
	}

	prompt := e.buildPrompt(task, currentNumeric, currentText, previousNumeric, rawSnapshot)

	resp, err := e.llmClient.ChatJSON(ctx, []llm.ChatMessage{
		{Role: "user", Content: prompt},
	})

	_ = e.budgetRepo.IncrementCallCount(ctx, task.ID)

	if err != nil {
		return false, fmt.Errorf("llm call: %w", err)
	}

	var result nlJudgeResponse
	if err := json.Unmarshal([]byte(resp), &result); err != nil {
		e.logger.Warn().Int64("task_id", task.ID).Str("response", resp).Msg("failed to parse nl_judge response")
		return false, nil
	}

	return result.ConditionMet, nil
}

func (e *NLJudgeEvaluator) buildPrompt(task domain.MonitorTask, currentNumeric *float64, currentText string, previousNumeric *float64, rawSnapshot string) string {
	var prevStr string
	if previousNumeric != nil {
		prevStr = fmt.Sprintf("%.4f", *previousNumeric)
	} else {
		prevStr = "none"
	}

	var curStr string
	if currentNumeric != nil {
		curStr = fmt.Sprintf("%.4f", *currentNumeric)
	} else {
		curStr = "none"
	}

	snapshot := rawSnapshot
	if len(snapshot) > 4000 {
		snapshot = snapshot[:4000]
	}

	return fmt.Sprintf(`Evaluate whether the following condition is met.

Condition: %s

Current numeric value: %s
Current text value: %s
Previous numeric value: %s

Raw data snapshot (may be truncated):
%s

Respond with a JSON object: {"condition_met": true/false, "reasoning": "brief explanation"}
Respond ONLY with the JSON object.`, task.ConditionPrompt, curStr, currentText, prevStr, snapshot)
}
