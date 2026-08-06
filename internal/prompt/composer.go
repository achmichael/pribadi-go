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
	
	// Build user prompt with history and dynamic instructions
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

// getClassificationDirectives returns classification-aware directives as a block.
func (c *composer) getClassificationDirectives(class *classifier.Classification) string {
	if class == nil {
		return ""
	}
	var additions []string
	
	switch class.Intent {
	case classifier.IntentCorrect:
		additions = append(additions, "<instruction> context: pengguna mengoreksi informasi sebelumnya. update pemahaman, jangan ulangi kesalahan. </instruction>")
	case classifier.IntentClarify:
		additions = append(additions, "<instruction> context: pengguna meminta klarifikasi. jelaskan lebih detail. </instruction>")
	}
	if class.RequiresClarify {
		additions = append(additions, "<instruction> warning: pertanyaan ambigu. minta klarifikasi jika tidak yakin. </instruction>")
	}
	
	if len(additions) > 0 {
		return strings.Join(additions, "\n")
	}
	return ""
}

// getPlanDirectives returns plan-aware directives as a block.
func (c *composer) getPlanDirectives(plan *planner.Plan) string {
	if plan == nil {
		return ""
	}
	var additions []string
	
	switch plan.ResponseStrategy {
	case "step_by_step":
		additions = append(additions, "<instruction> format: step-by-step </instruction>")
	case "comparison":
		additions = append(additions, "<instruction> format: perbandingan opsi (pro/kontra) </instruction>")
	case "summary":
		additions = append(additions, "<instruction> format: ringkasan komprehensif </instruction>")
	case "clarify_first":
		additions = append(additions, "<instruction> action: tanyakan klarifikasi sebelum menjawab </instruction>")
	}
	
	if plan.Reasoning != "" {
		additions = append(additions, fmt.Sprintf("<!-- Internal plan reasoning: %s -->", plan.Reasoning))
	}
	
	if len(additions) > 0 {
		return strings.Join(additions, "\n")
	}
	return ""
}

func (c *composer) buildUserPrompt(params ComposeParams) string {
	var sb strings.Builder
	sc := params.SessionContext

	hasContext := false

	// Inject dynamic instructions from Planner/Classifier
	classDirs := c.getClassificationDirectives(params.Classification)
	planDirs := c.getPlanDirectives(params.Plan)
	
	if classDirs != "" || planDirs != "" {
		if classDirs != "" {
			sb.WriteString(classDirs)
			sb.WriteString("\n")
		}
		if planDirs != "" {
			sb.WriteString(planDirs)
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	// Inject Memory Context
	if sc != nil && sc.MemoryContext != "" {
		if !hasContext {
			sb.WriteString("[Konteks Referensi]:\n")
			hasContext = true
		}
		sb.WriteString("Fakta Pengguna:\n")
		sb.WriteString(sc.MemoryContext)
		sb.WriteString("\n\n")
	}

	// Inject RAG Context
	if sc != nil && sc.HasRAG {
		if !hasContext {
			sb.WriteString("[Konteks Referensi]:\n")
			hasContext = true
		}
		sb.WriteString("Dokumen Relevan:\n")
		sb.WriteString(c.builder.truncateRAG(sc.RAGContext))
		sb.WriteString("\n")
		if len(sc.RAGSources) > 0 {
			sb.WriteString("Sumber: " + strings.Join(sc.RAGSources, ", ") + "\n")
		}
		sb.WriteString("\n")
	}

	if hasContext {
		sb.WriteString("[Pertanyaan Pengguna]:\n")
	}
	
	sb.WriteString(params.UserText)

	return sb.String()
}
