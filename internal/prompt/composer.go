// Package prompt provides enhanced prompt composition with Phase 2 intelligence.
package prompt

import (
	"context"
	"fmt"
	"strings"

	"github.com/achmichael/pribadi-go/internal/classifier"
	"github.com/achmichael/pribadi-go/internal/planner"
	"github.com/rs/zerolog"
)

// ─── Types ─────────────────────────────────────────────────────────

// ComposedPrompt is the full prompt ready for LLM, with metadata.
type ComposedPrompt struct {
	SystemPrompt string
	UserPrompt   string
	
	// Metadata about what was included
	IncludedRAG     bool
	IncludedMemory  bool
	IncludedHistory bool
	
	// Classification context
	MessageClass string
	Intent       string
	Plan         *planner.Plan
}

// ComposeParams holds inputs for prompt composition.
type ComposeParams struct {
	// User input
	UserText string
	
	// Classification from Phase 2
	Classification *classifier.Classification
	
	// Plan from Phase 2
	Plan *planner.Plan
	
	// Session context from context builder
	SessionContext *SessionContext
	
	// Conversation history (if needed by plan)
	ConversationHistory []HistoryMessage
}

// HistoryMessage represents one turn in conversation history.
type HistoryMessage struct {
	Role    string // "user" or "assistant"
	Content string
}

// ─── Interface ─────────────────────────────────────────────────────

// Composer assembles the final prompt with intelligence layer guidance.
type Composer interface {
	// Compose creates the final prompt based on classification and plan.
	Compose(ctx context.Context, params ComposeParams) (*ComposedPrompt, error)
}

// ─── Implementation ────────────────────────────────────────────────

type composer struct {
	builder *Builder
	logger  *zerolog.Logger
}

// NewComposer creates a Composer.
func NewComposer(builder *Builder, logger *zerolog.Logger) Composer {
	return &composer{
		builder: builder,
		logger:  logger,
	}
}

func (c *composer) Compose(ctx context.Context, params ComposeParams) (*ComposedPrompt, error) {
	if params.SessionContext == nil {
		return nil, fmt.Errorf("session context required")
	}

	sc := params.SessionContext
	
	// Apply plan filtering to context
	if params.Plan != nil {
		sc = c.applyPlanFilters(sc, params.Plan)
	}
	
	// Build base system prompt
	systemPrompt := c.builder.Build(*sc)
	
	// Enhance with classification-specific directives
	if params.Classification != nil {
		systemPrompt = c.enhanceWithClassification(systemPrompt, params.Classification)
	}
	
	// Enhance with plan-specific instructions
	if params.Plan != nil {
		systemPrompt = c.enhanceWithPlan(systemPrompt, params.Plan)
	}
	
	// Build user prompt with history if needed
	userPrompt := c.buildUserPrompt(params)
	
	result := &ComposedPrompt{
		SystemPrompt:    systemPrompt,
		UserPrompt:      userPrompt,
		IncludedRAG:     sc.HasRAG,
		IncludedMemory:  sc.MemoryContext != "",
		IncludedHistory: len(params.ConversationHistory) > 0,
		Plan:            params.Plan,
	}
	
	if params.Classification != nil {
		result.MessageClass = string(params.Classification.MessageClass)
		result.Intent = string(params.Classification.Intent)
	}
	
	c.logger.Debug().
		Str("class", result.MessageClass).
		Str("intent", result.Intent).
		Bool("rag", result.IncludedRAG).
		Bool("memory", result.IncludedMemory).
		Bool("history", result.IncludedHistory).
		Msg("[composer] prompt composed")
	
	return result, nil
}

// applyPlanFilters removes context that the plan says isn't needed.
func (c *composer) applyPlanFilters(sc *SessionContext, plan *planner.Plan) *SessionContext {
	filtered := *sc // copy
	
	if !plan.NeedsRAG {
		filtered.RAGContext = ""
		filtered.HasRAG = false
		filtered.RAGSources = nil
	}
	
	if !plan.NeedsMemory {
		filtered.MemoryContext = ""
	}
	
	if !plan.NeedsDocContext {
		filtered.MetadataBlock = ""
		filtered.TargetDocID = ""
	}
	
	return &filtered
}

// enhanceWithClassification adds classification-aware directives.
func (c *composer) enhanceWithClassification(sysPrompt string, class *classifier.Classification) string {
	var additions []string
	
	switch class.Intent {
	case classifier.IntentCorrect:
		additions = append(additions, "\n--- KOREKSI PENGGUNA ---")
		additions = append(additions, "Pengguna mengoreksi informasi sebelumnya. Terima koreksi dan update pemahaman Anda.")
		additions = append(additions, "Jangan ulangi kesalahan yang sama.")
		additions = append(additions, "--- END KOREKSI ---\n")
		
	case classifier.IntentClarify:
		additions = append(additions, "\n--- KLARIFIKASI ---")
		additions = append(additions, "Pengguna meminta klarifikasi. Jelaskan dengan lebih detail dan jelas.")
		additions = append(additions, "--- END KLARIFIKASI ---\n")
		
	}
	if class.RequiresClarify {
		additions = append(additions, "\n--- PERINGATAN AMBIGUITAS ---")
		additions = append(additions, "Pertanyaan pengguna mungkin ambigu. Jika tidak yakin, minta klarifikasi sebelum menjawab.")
		additions = append(additions, "--- END PERINGATAN ---\n")
	}
	
	if len(additions) > 0 {
		return sysPrompt + "\n" + strings.Join(additions, "\n")
	}
	return sysPrompt
}

// enhanceWithPlan adds plan-aware directives.
func (c *composer) enhanceWithPlan(sysPrompt string, plan *planner.Plan) string {
	var additions []string
	
	switch plan.ResponseStrategy {
	case "step_by_step":
		additions = append(additions, "\n--- STRATEGI RESPONS ---")
		additions = append(additions, "Jawab dengan pendekatan step-by-step. Nomori setiap langkah.")
		additions = append(additions, "--- END STRATEGI ---\n")
		
	case "comparison":
		additions = append(additions, "\n--- STRATEGI RESPONS ---")
		additions = append(additions, "Jawab dengan membandingkan opsi-opsi yang ada. Jelaskan pro-kontra masing-masing.")
		additions = append(additions, "--- END STRATEGI ---\n")
		
	case "summary":
		additions = append(additions, "\n--- STRATEGI RESPONS ---")
		additions = append(additions, "Berikan ringkasan (summary) yang komprehensif. Fokus pada poin-poin utama.")
		additions = append(additions, "--- END STRATEGI ---\n")
		
	case "clarify_first":
		additions = append(additions, "\n--- STRATEGI RESPONS ---")
		additions = append(additions, "Tanyakan klarifikasi SEBELUM menjawab. Pertanyaan pengguna memerlukan informasi tambahan.")
		additions = append(additions, "--- END STRATEGI ---\n")
	}
	
	if plan.Reasoning != "" {
		additions = append(additions, fmt.Sprintf("\n<!-- Internal plan reasoning: %s -->", plan.Reasoning))
	}
	
	if len(additions) > 0 {
		return sysPrompt + "\n" + strings.Join(additions, "\n")
	}
	return sysPrompt
}

func (c *composer) buildUserPrompt(params ComposeParams) string {
	return params.UserText
}
