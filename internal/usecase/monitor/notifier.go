package monitor

import (
	"context"
	"fmt"

	"github.com/achmichael/pribadi-go/internal/domain"
	"github.com/achmichael/pribadi-go/pkg/llm"
)

type Notifier struct {
	llmClient llm.Client
}

func NewNotifier(llmClient llm.Client) *Notifier {
	return &Notifier{llmClient: llmClient}
}

func (n *Notifier) GenerateMessage(ctx context.Context, task domain.MonitorTask, numeric *float64, text string) (string, error) {
	var valStr string
	if numeric != nil {
		valStr = fmt.Sprintf("%.4f", *numeric)
	} else {
		valStr = text
	}

	prompt := fmt.Sprintf(`Generate a short notification message (1-3 sentences) for the user.

Monitor name: %s
Category: %s
Current value: %s
Condition that was met: %s

Write a clear, concise notification. Do not use markdown. Just plain text.`,
		task.Name, task.Category, valStr, n.conditionDesc(task))

	msg, err := n.llmClient.SimplyChat(ctx, prompt)
	if err != nil {
		return fmt.Sprintf("[Monitor Alert] %s: value is %s", task.Name, valStr), nil
	}
	return msg, nil
}

func (n *Notifier) conditionDesc(task domain.MonitorTask) string {
	if task.ConditionMode == domain.ConditionNLJudge {
		return task.ConditionPrompt
	}
	switch task.Operator {
	case domain.OpAbove:
		return fmt.Sprintf("value above %.4f", task.ConditionValue)
	case domain.OpBelow:
		return fmt.Sprintf("value below %.4f", task.ConditionValue)
	case domain.OpPercentChange:
		return fmt.Sprintf("change >= %.2f%%", task.ConditionValue)
	case domain.OpEquals:
		if task.ConditionText != "" {
			return fmt.Sprintf("equals %q", task.ConditionText)
		}
		return fmt.Sprintf("equals %.4f", task.ConditionValue)
	case domain.OpContains:
		return fmt.Sprintf("contains %q", task.ConditionText)
	default:
		return string(task.Operator)
	}
}
