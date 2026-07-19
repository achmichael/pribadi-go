package reasoning

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/achmichael/pribadi-go/pkg/ollama"
	"github.com/rs/zerolog"
)

// ─── Types ─────────────────────────────────────────────────────────

// Reflection represents post-response analysis.
type Reflection struct {
	WhatWorked      []string `json:"what_worked"`
	WhatFailed      []string `json:"what_failed"`
	MissedContext   []string `json:"missed_context"`
	ShouldImprove   []string `json:"should_improve"`
	ConfidenceLevel string   `json:"confidence_level"` // "high", "medium", "low"
	NeedsFollowup   bool     `json:"needs_followup"`
	FollowupReason  string   `json:"followup_reason"`
}

// ReflectionParams holds inputs for reflection.
type ReflectionParams struct {
	UserQuestion string
	Response     string
	Plan         string // the plan that was executed
	RAGUsed      bool
	MemoryUsed   bool
	HistoryUsed  bool
	UserFeedback string // if user provided immediate feedback
}

// ─── Interface ─────────────────────────────────────────────────────

// Reflector analyzes response quality and learns from execution.
type Reflector interface {
	// Reflect performs post-response analysis.
	Reflect(ctx context.Context, params ReflectionParams) (*Reflection, error)
}

// ─── Implementation ────────────────────────────────────────────────

type reflector struct {
	llm    *ollama.OllamaClient
	logger *zerolog.Logger
}

// NewReflector creates a Reflector.
func NewReflector(llm *ollama.OllamaClient, logger *zerolog.Logger) Reflector {
	return &reflector{
		llm:    llm,
		logger: logger,
	}
}

func (r *reflector) Reflect(ctx context.Context, params ReflectionParams) (*Reflection, error) {
	start := time.Now()

	// Skip reflection for very short interactions
	if len(params.Response) < 30 {
		r.logger.Debug().Msg("[reflector] skipping reflection for short response")
		return &Reflection{
			WhatWorked:      []string{"Response delivered"},
			ConfidenceLevel: "high",
		}, nil
	}

	prompt := r.buildReflectionPrompt(params)

	messages := []ollama.ChatMessage{
		{Role: "user", Content: prompt},
	}

	reply, err := r.llm.ChatJSON(ctx, messages)
	if err != nil {
		r.logger.Warn().Err(err).Msg("[reflector] reflection failed")
		return r.fallbackReflection(), nil
	}

	var result Reflection
	if err := json.Unmarshal([]byte(strings.TrimSpace(reply)), &result); err != nil {
		r.logger.Warn().Err(err).Msg("[reflector] parse reflection failed")
		return r.fallbackReflection(), nil
	}

	r.logger.Info().
		Str("confidence", result.ConfidenceLevel).
		Int("worked", len(result.WhatWorked)).
		Int("failed", len(result.WhatFailed)).
		Bool("needs_followup", result.NeedsFollowup).
		Dur("ms", time.Since(start)).
		Msg("[reflector] reflection complete")

	return &result, nil
}

func (r *reflector) buildReflectionPrompt(params ReflectionParams) string {
	var sb strings.Builder

	sb.WriteString("You are a reflection agent. Analyze the quality of this response execution.\n\n")

	sb.WriteString(fmt.Sprintf("USER QUESTION: %s\n\n", params.UserQuestion))
	sb.WriteString(fmt.Sprintf("RESPONSE GIVEN: %s\n\n", params.Response))
	sb.WriteString(fmt.Sprintf("PLAN EXECUTED: %s\n\n", params.Plan))

	sb.WriteString("RESOURCES USED:\n")
	sb.WriteString(fmt.Sprintf("- RAG/Documents: %v\n", params.RAGUsed))
	sb.WriteString(fmt.Sprintf("- User Memory: %v\n", params.MemoryUsed))
	sb.WriteString(fmt.Sprintf("- Conversation History: %v\n\n", params.HistoryUsed))

	if params.UserFeedback != "" {
		sb.WriteString(fmt.Sprintf("USER FEEDBACK: %s\n\n", params.UserFeedback))
	}

	sb.WriteString(`REFLECT ON:
1. What worked well in this response?
2. What failed or could be improved?
3. Was any important context missing?
4. Should we follow up with the user?

Return ONLY valid JSON:
{
  "what_worked": ["things that went well"],
  "what_failed": ["things that didn't work"],
  "missed_context": ["context we should have used"],
  "should_improve": ["specific improvements for next time"],
  "confidence_level": "high/medium/low",
  "needs_followup": true/false,
  "followup_reason": "why followup is needed"
}`)

	return sb.String()
}

func (r *reflector) fallbackReflection() *Reflection {
	return &Reflection{
		WhatWorked:      []string{"Response generated"},
		ConfidenceLevel: "medium",
		NeedsFollowup:   false,
	}
}
