// Package planner implements pre-generation reasoning.
// Before the LLM generates a response, the planner determines
// what information is needed and how to structure the approach
// (Principle 18: planning step).
package planner

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/achmichael/pribadi-go/internal/classifier"
	"github.com/achmichael/pribadi-go/internal/domain"
	"github.com/achmichael/pribadi-go/pkg/ollama"
	"github.com/rs/zerolog"
)

// ─── Types ─────────────────────────────────────────────────────────

// Plan describes what the orchestrator should do for this turn.
type Plan struct {
	// NeedsRAG indicates whether RAG retrieval is needed.
	NeedsRAG bool `json:"needs_rag"`

	// NeedsMemory indicates whether fact memory lookup is needed.
	NeedsMemory bool `json:"needs_memory"`

	// NeedsHistory indicates whether conversation history is needed.
	NeedsHistory bool `json:"needs_history"`

	// NeedsDocContext indicates whether document context is relevant.
	NeedsDocContext bool `json:"needs_doc_context"`

	// ResponseStrategy describes how to answer.
	ResponseStrategy string `json:"response_strategy"` // "direct", "step_by_step", "comparison", "summary", "clarify_first"

	// Reasoning is internal reasoning about why this plan was chosen.
	Reasoning string `json:"reasoning"`

	// Confidence in this plan.
	Confidence string `json:"confidence"`
}

// ─── Interface ─────────────────────────────────────────────────────

// Planner creates an execution plan for each turn.
type Planner interface {
	// CreatePlan analyzes the classified message and determines what's needed.
	CreatePlan(ctx context.Context, params PlanParams) (*Plan, error)
}

// PlanParams holds inputs for planning.
type PlanParams struct {
	UserText       string
	Classification *classifier.Classification
	State          *domain.StateData
	HasActiveDoc   bool
	TurnCount      int
}

// ─── Implementation ────────────────────────────────────────────────

type planner struct {
	llm    *ollama.OllamaClient
	logger *zerolog.Logger
}

// NewPlanner creates a Planner.
func NewPlanner(llm *ollama.OllamaClient, logger *zerolog.Logger) Planner {
	return &planner{
		llm:    llm,
		logger: logger,
	}
}

func (p *planner) CreatePlan(ctx context.Context, params PlanParams) (*Plan, error) {
	start := time.Now()

	// Fast path: use heuristics for clear cases
	if plan := p.heuristicPlan(params); plan != nil {
		p.logger.Debug().
			Str("strategy", plan.ResponseStrategy).
			Bool("rag", plan.NeedsRAG).
			Bool("memory", plan.NeedsMemory).
			Str("method", "heuristic").
			Dur("ms", time.Since(start)).
			Msg("[planner] plan created")
		return plan, nil
	}

	// LLM-based planning for complex cases
	plan, err := p.llmPlan(ctx, params)
	if err != nil {
		p.logger.Warn().Err(err).Msg("[planner] LLM plan failed, using fallback")
		plan = p.fallbackPlan(params)
	}

	p.logger.Info().
		Str("strategy", plan.ResponseStrategy).
		Bool("rag", plan.NeedsRAG).
		Bool("memory", plan.NeedsMemory).
		Bool("doc", plan.NeedsDocContext).
		Str("confidence", plan.Confidence).
		Dur("ms", time.Since(start)).
		Msg("[planner] plan created")

	return plan, nil
}

// heuristicPlan handles obvious cases without LLM.
func (p *planner) heuristicPlan(params PlanParams) *Plan {
	if params.Classification == nil {
		return nil
	}

	switch params.Classification.Intent {
	case classifier.IntentChitChat:
		return &Plan{
			NeedsRAG:         false,
			NeedsMemory:      true, // use memory for personalization
			NeedsHistory:     true, // need history for context
			NeedsDocContext:  false,
			ResponseStrategy: "direct",
			Reasoning:        "Casual conversation, no document/RAG/Memory needed",
			Confidence:       "high",
		}

	case classifier.IntentSetPreference:
		return &Plan{
			NeedsRAG:         false,
			NeedsMemory:      false,
			NeedsHistory:     false,
			NeedsDocContext:  false,
			ResponseStrategy: "direct",
			Reasoning:        "Preference update, just acknowledge",
			Confidence:       "high",
		}

	case classifier.IntentAskDocument:
		return &Plan{
			NeedsRAG:         true,
			NeedsMemory:      false,
			NeedsHistory:     true,
			NeedsDocContext:  true,
			ResponseStrategy: "direct",
			Reasoning:        "Document question, needs RAG + doc context",
			Confidence:       "high",
		}

	case classifier.IntentCorrect:
		return &Plan{
			NeedsRAG:         false,
			NeedsMemory:      true,
			NeedsHistory:     true,
			NeedsDocContext:  params.HasActiveDoc,
			ResponseStrategy: "direct",
			Reasoning:        "Correction, needs history to understand what was wrong",
			Confidence:       "high",
		}
	}

	return nil
}

// llmPlan uses LLM for complex planning.
func (p *planner) llmPlan(ctx context.Context, params PlanParams) (*Plan, error) {
	classJSON := "{}"
	if params.Classification != nil {
		b, _ := json.Marshal(params.Classification)
		classJSON = string(b)
	}

	prompt := fmt.Sprintf(`You are a planning agent. Given a classified user message, determine what resources are needed to answer it.

CLASSIFICATION: %s
USER MESSAGE: %s
HAS ACTIVE DOCUMENT: %v
TURN COUNT: %d

Return ONLY valid JSON:
{
  "needs_rag": true/false,
  "needs_memory": true/false,
  "needs_history": true/false,
  "needs_doc_context": true/false,
  "response_strategy": "direct|step_by_step|comparison|summary|clarify_first",
  "reasoning": "brief explanation",
  "confidence": "high|medium|low"
}`, classJSON, params.UserText, params.HasActiveDoc, params.TurnCount)

	messages := []ollama.ChatMessage{
		{Role: "user", Content: prompt},
	}

	reply, err := p.llm.ChatJSON(ctx, messages)
	if err != nil {
		return nil, err
	}

	var plan Plan
	if err := json.Unmarshal([]byte(strings.TrimSpace(reply)), &plan); err != nil {
		return nil, fmt.Errorf("parse plan: %w", err)
	}

	return &plan, nil
}

// fallbackPlan provides safe defaults.
func (p *planner) fallbackPlan(params PlanParams) *Plan {
	return &Plan{
		NeedsRAG:         true,
		NeedsMemory:      true,
		NeedsHistory:     true,
		NeedsDocContext:  params.HasActiveDoc,
		ResponseStrategy: "direct",
		Reasoning:        "fallback: retrieve everything",
		Confidence:       "low",
	}
}
