// Package response handles final response assembly and post-processing.
// After LLM generation and verification, this layer updates state,
// triggers memory sync, formats citations, and logs the interaction.
package response

import (
	"context"
	"fmt"
	"time"

	"github.com/achmichael/pribadi-go/internal/classifier"
	"github.com/achmichael/pribadi-go/internal/conversation"
	"github.com/achmichael/pribadi-go/internal/domain"
	"github.com/achmichael/pribadi-go/internal/factmemory"
	"github.com/achmichael/pribadi-go/internal/planner"
	"github.com/achmichael/pribadi-go/internal/prompt"
	"github.com/achmichael/pribadi-go/internal/reasoning"
	"github.com/achmichael/pribadi-go/internal/repository"
	"github.com/rs/zerolog"
)

// ─── Types ─────────────────────────────────────────────────────────

// FinalResponse is the complete response ready to send to the user.
type FinalResponse struct {
	Text         string
	Citations    []prompt.Citation
	Metadata     ResponseMetadata
	ShouldUpdate bool // whether to update state/memory
}

// ResponseMetadata holds telemetry about response generation.
type ResponseMetadata struct {
	Classification *classifier.Classification
	Plan           *planner.Plan
	Verification   *reasoning.VerificationResult
	Reflection     *reasoning.Reflection
	
	// Timing
	TotalDurationMs    int64
	ClassificationMs   int64
	PlanningMs         int64
	ContextBuildMs     int64
	LLMInferenceMs     int64
	VerificationMs     int64
	ReflectionMs       int64
	
	// Resources used
	RAGUsed     bool
	MemoryUsed  bool
	HistoryUsed bool
	
	// Quality metrics
	ConfidenceScore float32
	HallucinationRisk string
}

// ProcessParams holds inputs for response processing.
type ProcessParams struct {
	UserID       string
	SessionID    string
	UserMessage  string
	RawResponse  string
	
	// Phase outputs
	Classification *classifier.Classification
	Plan           *planner.Plan
	ComposedPrompt *prompt.ComposedPrompt
	Verification   *reasoning.VerificationResult
	
	// Context
	RAGSources []string
	State      *domain.StateData
}

// ─── Interface ─────────────────────────────────────────────────────

// Processor handles final response assembly and post-processing.
type Processor interface {
	// Process assembles final response with citations and metadata.
	Process(ctx context.Context, params ProcessParams) (*FinalResponse, error)
	
	// UpdateState persists conversation state after response.
	UpdateState(ctx context.Context, userID, sessionID string, classification *classifier.Classification) error
	
	// TriggerMemorySync schedules async memory extraction.
	TriggerMemorySync(userID, userMessage, assistantMessage string)
}

// ─── Implementation ────────────────────────────────────────────────

type processor struct {
	stateManager conversation.StateManager
	memory       factmemory.MemoryManager
	repo         repository.Repository
	citFormatter prompt.CitationFormatter
	logger       *zerolog.Logger
}

// NewProcessor creates a Processor.
func NewProcessor(
	stateManager conversation.StateManager,
	memory factmemory.MemoryManager,
	repo repository.Repository,
	logger *zerolog.Logger,
) Processor {
	return &processor{
		stateManager: stateManager,
		memory:       memory,
		repo:         repo,
		citFormatter: prompt.NewCitationFormatter(),
		logger:       logger,
	}
}

func (p *processor) Process(ctx context.Context, params ProcessParams) (*FinalResponse, error) {
	start := time.Now()
	
	// Build citations from RAG sources
	citations := p.buildCitations(params.RAGSources)
	
	// Apply citation formatting
	responseText := params.RawResponse
	if len(citations) > 0 {
		citationStyle := p.determineCitationStyle(params.State)
		responseText = p.citFormatter.Format(responseText, citations, citationStyle)
	}
	
	// Build metadata
	metadata := ResponseMetadata{
		Classification: params.Classification,
		Plan:           params.Plan,
		Verification:   params.Verification,
		RAGUsed:        params.ComposedPrompt != nil && params.ComposedPrompt.IncludedRAG,
		MemoryUsed:     params.ComposedPrompt != nil && params.ComposedPrompt.IncludedMemory,
		HistoryUsed:    params.ComposedPrompt != nil && params.ComposedPrompt.IncludedHistory,
	}
	
	if params.Verification != nil {
		metadata.ConfidenceScore = params.Verification.ConfidenceScore
		metadata.HallucinationRisk = params.Verification.HallucinationRisk
	}

	// ─── Post-generation Grounding Check (Safety Net) ───
	if params.ComposedPrompt != nil && params.ComposedPrompt.IncludedRAG {
		// Rely on verification result if present
		if params.Verification != nil {
			if !params.Verification.IsValid {
				p.logger.Warn().
					Str("user_id", params.UserID).
					Str("session_id", params.SessionID).
					Float32("confidence", metadata.ConfidenceScore).
					Msg("[grounding] entity mismatch or hallucination risk detected by verifier")
				
				metadata.ConfidenceScore = 0.5
				metadata.HallucinationRisk = "medium"
			}
		}
	}
	
	// Determine if state/memory should update
	shouldUpdate := true
	if params.Classification != nil {
		// Don't update for simple greetings or pure instructions
		if params.Classification.Intent == classifier.IntentChitChat && len(params.RawResponse) < 50 {
			shouldUpdate = false
		}
	}
	
	result := &FinalResponse{
		Text:         responseText,
		Citations:    citations,
		Metadata:     metadata,
		ShouldUpdate: shouldUpdate,
	}
	
	p.logger.Info().
		Int("citations", len(citations)).
		Bool("should_update", shouldUpdate).
		Float32("confidence", metadata.ConfidenceScore).
		Dur("processing_ms", time.Since(start)).
		Msg("[response] processed")
	
	return result, nil
}

func (p *processor) UpdateState(ctx context.Context, userID, sessionID string, classification *classifier.Classification) error {
	// Load current state
	state, err := p.stateManager.Load(ctx, userID, sessionID)
	if err != nil {
		p.logger.Warn().Err(err).Msg("[response] state load failed for update")
		defaultState := domain.DefaultStateData()
		state = &defaultState
	}
	
	// Determine intent and class for state update
	intent := ""
	class := ""
	if classification != nil {
		intent = string(classification.Intent)
		class = string(classification.MessageClass)
	}
	
	// Update state (increments turn count)
	if err := p.stateManager.Update(ctx, userID, sessionID, state, intent, class, ""); err != nil {
		return fmt.Errorf("update state: %w", err)
	}
	
	p.logger.Debug().
		Str("user_id", userID).
		Str("session_id", sessionID).
		Str("intent", intent).
		Msg("[response] state updated")
	
	return nil
}

func (p *processor) TriggerMemorySync(userID, userMessage, assistantMessage string) {
	// Async memory extraction - non-blocking
	p.memory.SyncAsync(userID, userMessage, assistantMessage)
	
	p.logger.Debug().
		Str("user_id", userID).
		Msg("[response] memory sync triggered")
}

// ─── Helpers ───────────────────────────────────────────────────────

func (p *processor) buildCitations(sources []string) []prompt.Citation {
	if len(sources) == 0 {
		return nil
	}
	
	// Deduplicate citations by source ID
	seen := make(map[string]bool)
	var citations []prompt.Citation
	
	idx := 1
	for _, src := range sources {
		if seen[src] {
			continue
		}
		seen[src] = true
		citations = append(citations, prompt.Citation{
			Number: idx,
			Source: src,
			Type:   "document",
		})
		idx++
	}
	return citations
}

func (p *processor) determineCitationStyle(state *domain.StateData) prompt.CitationStyle {
	if state == nil {
		return prompt.CitationInline
	}
	
	// Check custom directives
	if state.CustomDirectives != nil {
		if style, ok := state.CustomDirectives["citation_style"]; ok {
			switch style {
			case "footnote":
				return prompt.CitationFootnote
			case "none":
				return prompt.CitationNone
			case "inline":
				return prompt.CitationInline
			}
		}
	}
	
	// Default to inline
	return prompt.CitationNone
}
