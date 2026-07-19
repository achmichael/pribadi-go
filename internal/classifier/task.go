package classifier

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/achmichael/pribadi-go/internal/repository"
	"github.com/achmichael/pribadi-go/pkg/ollama"
	"github.com/rs/zerolog"
)

// ─── Interface ─────────────────────────────────────────────────────

// TaskClassifier detects whether a message continues an existing task
// or introduces a new topic (Principle 15).
type TaskClassifier interface {
	// Classify determines if the message continues an active task or starts new.
	Classify(ctx context.Context, params TaskClassifyParams) (*TaskClassification, error)
}

// TaskClassifyParams holds inputs for task classification.
type TaskClassifyParams struct {
	UserText    string
	ActiveTask  string // description of current task, empty if none
	LastIntent  string
	SessionID   string
	UserID      string
	TurnCount   int
}

// TaskClassification is the result.
type TaskClassification struct {
	ContinuesTask bool   `json:"continues_task"`
	NewTaskDesc   string `json:"new_task_description"` // if new task detected
	TopicShift    bool   `json:"topic_shift"`          // clear topic change
	Confidence    string `json:"confidence"`
}

// ─── Implementation ────────────────────────────────────────────────

type taskClassifier struct {
	llm    *ollama.OllamaClient
	repo   repository.Repository
	logger *zerolog.Logger
}

// NewTaskClassifier creates a TaskClassifier.
func NewTaskClassifier(llm *ollama.OllamaClient, repo repository.Repository, logger *zerolog.Logger) TaskClassifier {
	return &taskClassifier{
		llm:    llm,
		repo:   repo,
		logger: logger,
	}
}

func (c *taskClassifier) Classify(ctx context.Context, params TaskClassifyParams) (*TaskClassification, error) {
	start := time.Now()

	// No active task → always new
	if params.ActiveTask == "" {
		result := &TaskClassification{
			ContinuesTask: false,
			NewTaskDesc:   c.summarizeTask(params.UserText),
			TopicShift:    false,
			Confidence:    "high",
		}
		c.logger.Debug().
			Str("result", "new_task").
			Dur("ms", time.Since(start)).
			Msg("[task_classifier] no active task")
		return result, nil
	}

	// First turn → always new
	if params.TurnCount <= 1 {
		return &TaskClassification{
			ContinuesTask: false,
			NewTaskDesc:   c.summarizeTask(params.UserText),
			Confidence:    "high",
		}, nil
	}

	// Heuristic: very short follow-ups likely continue
	text := strings.TrimSpace(params.UserText)
	if len(text) < 30 && !strings.Contains(text, "?") {
		// Short non-question → likely continuation
		return &TaskClassification{
			ContinuesTask: true,
			Confidence:    "medium",
		}, nil
	}

	// Fetch recent history for context
	history := ""
	if msgs, err := c.repo.ListMessagesByUserSession(ctx, params.UserID, params.SessionID, 4); err == nil {
		var sb strings.Builder
		for _, m := range msgs {
			content := m.Content
			if len(content) > 200 {
				content = content[:200] + "..."
			}
			sb.WriteString(fmt.Sprintf("%s: %s\n", strings.ToUpper(m.Role), content))
		}
		history = sb.String()
	}

	prompt := fmt.Sprintf(`Analyze whether this message continues the active task or introduces a new topic.

ACTIVE TASK: %s
RECENT HISTORY:
%s
NEW MESSAGE: %s

Return ONLY valid JSON:
{
  "continues_task": true/false,
  "new_task_description": "brief description if new task, empty if continues",
  "topic_shift": true/false,
  "confidence": "high/medium/low"
}`, params.ActiveTask, history, params.UserText)

	messages := []ollama.ChatMessage{
		{Role: "user", Content: prompt},
	}

	reply, err := c.llm.ChatJSON(ctx, messages)
	if err != nil {
		c.logger.Warn().Err(err).Msg("[task_classifier] LLM failed, assuming continuation")
		return &TaskClassification{
			ContinuesTask: true,
			Confidence:    "low",
		}, nil
	}

	var result TaskClassification
	if err := json.Unmarshal([]byte(strings.TrimSpace(reply)), &result); err != nil {
		c.logger.Warn().Err(err).Str("reply", reply).Msg("[task_classifier] parse failed")
		return &TaskClassification{
			ContinuesTask: true,
			Confidence:    "low",
		}, nil
	}

	c.logger.Info().
		Bool("continues", result.ContinuesTask).
		Bool("topic_shift", result.TopicShift).
		Str("confidence", result.Confidence).
		Dur("ms", time.Since(start)).
		Msg("[task_classifier] classified")

	return &result, nil
}

// summarizeTask creates a brief task description from user text.
func (c *taskClassifier) summarizeTask(text string) string {
	if len(text) <= 80 {
		return text
	}
	// Take first 80 chars + ellipsis
	words := strings.Fields(text)
	var sb strings.Builder
	for _, w := range words {
		if sb.Len()+len(w)+1 > 80 {
			break
		}
		if sb.Len() > 0 {
			sb.WriteString(" ")
		}
		sb.WriteString(w)
	}
	sb.WriteString("...")
	return sb.String()
}
