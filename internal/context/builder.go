// Package context assembles all context sources needed for a single
// LLM inference turn: conversation state, user preferences, RAG results,
// fact memory, document metadata, and conversation history.
package context

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/achmichael/pribadi-go/internal/conversation"
	"github.com/achmichael/pribadi-go/internal/domain"
	"github.com/achmichael/pribadi-go/internal/factmemory"
	"github.com/achmichael/pribadi-go/internal/prompt"
	"github.com/achmichael/pribadi-go/internal/repository"
	"github.com/achmichael/pribadi-go/internal/usecase/rag"
	"github.com/achmichael/pribadi-go/pkg/ollama"
	"github.com/rs/zerolog"
)

// ─── Interface ─────────────────────────────────────────────────────

// Builder assembles SessionContext from all sources.
type Builder interface {
	// Build gathers all context for a single inference turn.
	Build(ctx context.Context, params BuildParams) (*prompt.SessionContext, error)
}

// BuildParams holds inputs needed to build context.
type BuildParams struct {
	UserID    string
	SessionID string

	// User's text after preprocessing (transcription/OCR done)
	UserText string

	// RAG query (may be enriched by reference resolver)
	RAGQuery    string
	TargetDocID string

	// Conversation history messages to include
	HistoryLimit int

	// Plan-based filtering (Phase 2)
	SkipRAG    bool
	SkipMemory bool
}

// ─── Implementation ────────────────────────────────────────────────

type builder struct {
	stateManager conversation.StateManager
	prefManager  conversation.PreferenceManager
	memory       factmemory.MemoryManager
	ragRetrieve  rag.RetrievalService
	repo         repository.Repository
	dashRepo     repository.DashboardRepository
	llm          *ollama.OllamaClient
	logger       *zerolog.Logger
}

// NewBuilder creates a context Builder.
func NewBuilder(
	stateManager conversation.StateManager,
	prefManager conversation.PreferenceManager,
	memory factmemory.MemoryManager,
	ragRetrieve rag.RetrievalService,
	repo repository.Repository,
	dashRepo repository.DashboardRepository,
	llm *ollama.OllamaClient,
	logger *zerolog.Logger,
) Builder {
	return &builder{
		stateManager: stateManager,
		prefManager:  prefManager,
		memory:       memory,
		ragRetrieve:  ragRetrieve,
		repo:         repo,
		dashRepo:     dashRepo,
		llm:          llm,
		logger:       logger,
	}
}

func (b *builder) Build(ctx context.Context, params BuildParams) (*prompt.SessionContext, error) {
	start := time.Now()

	// 1. Load conversation state
	state, err := b.stateManager.Load(ctx, params.UserID, params.SessionID)
	if err != nil {
		b.logger.Warn().Err(err).Msg("[context] state load failed, using defaults")
		defaultState := domain.DefaultStateData()
		state = &defaultState
	}

	// 2. Load persona from dashboard config
	persona := b.loadPersona(ctx)

	// 3. Load prompt template
	promptTemplate := b.loadPromptTemplate(ctx)

	// 4. Parallel: RAG retrieval + memory prefetch
	type ragResult struct {
		ctx rag.PromptContext
		err error
	}
	type memResult struct {
		context string
		err     error
	}

	ragCh := make(chan ragResult, 1)
	memCh := make(chan memResult, 1)

	// RAG retrieval (skip if plan says not needed)
	if params.SkipRAG {
		b.logger.Debug().Msg("[context] skipping RAG per plan")
		ragCh <- ragResult{ctx: rag.PromptContext{HasResults: false}, err: nil}
	} else {
		go func() {
			retStart := time.Now()
			ctxInfo, err := b.ragRetrieve.Retrieve(ctx, params.RAGQuery, params.TargetDocID)
			b.logger.Info().
				Bool("has_results", ctxInfo.HasResults).
				Int("context_len", len(ctxInfo.Context)).
				Dur("ms", time.Since(retStart)).
				Msg("[context] RAG retrieval done")
			ragCh <- ragResult{ctx: ctxInfo, err: err}
		}()
	}

	// Memory prefetch (skip if plan says not needed)
	if params.SkipMemory {
		b.logger.Debug().Msg("[context] skipping memory per plan")
		memCh <- memResult{context: "", err: nil}
	} else {
		go func() {
			memStart := time.Now()
			memCtx, err := b.memory.PrefetchRelevant(ctx, params.UserID, params.UserText)
			b.logger.Info().
				Int("memory_len", len(memCtx)).
				Dur("ms", time.Since(memStart)).
				Msg("[context] memory prefetch done")
			memCh <- memResult{context: memCtx, err: err}
		}()
	}

	ragRes := <-ragCh
	memRes := <-memCh

	ctxInfo := ragRes.ctx
	if ragRes.err != nil {
		b.logger.Warn().Err(ragRes.err).Msg("[context] RAG retrieval failed")
		ctxInfo = rag.PromptContext{HasResults: false}
	}

	memoryContext := ""
	if memRes.err != nil {
		b.logger.Warn().Err(memRes.err).Msg("[context] memory prefetch failed")
	} else {
		memoryContext = memRes.context
	}

	// 5. Build document metadata block if targeting a document
	metadataBlock := ""
	if params.TargetDocID != "" {
		metadataBlock = b.buildMetadataBlock(ctx, params.TargetDocID)
	}

	// 6. Build preferences block
	prefsBlock := b.buildPreferencesBlock(ctx, params.UserID)

	// 7. Get turn count
	turnCount := 0
	if raw, err := b.stateManager.GetRaw(ctx, params.UserID, params.SessionID); err == nil && raw != nil {
		turnCount = raw.TurnCount
	}

	sc := &prompt.SessionContext{
		State:            state,
		TurnCount:        turnCount,
		Persona:          persona,
		PromptTemplate:   promptTemplate,
		RAGContext:       ctxInfo.Context,
		HasRAG:           ctxInfo.HasResults,
		RAGSources:       ctxInfo.Sources,
		MemoryContext:    memoryContext,
		MetadataBlock:    metadataBlock,
		TargetDocID:      params.TargetDocID,
		PreferencesBlock: prefsBlock,
	}

	b.logger.Info().
		Dur("total_ms", time.Since(start)).
		Bool("has_rag", sc.HasRAG).
		Bool("has_memory", memoryContext != "").
		Bool("has_doc", params.TargetDocID != "").
		Int("turn", turnCount).
		Msg("[context] build complete")

	return sc, nil
}

// ─── Helpers ───────────────────────────────────────────────────────

func (b *builder) loadPersona(ctx context.Context) prompt.Persona {
	p := prompt.Persona{
		Name:            "Assistant",
		Description:     "Saya adalah asisten AI yang cerdas dan efisien.",
		Tone:            "casual",
		VoiceGuidelines: "Gunakan bahasa natural dan hangat, TAPI tidak berlebihan (hindari \"Haha!\", \"Asyik!\", \"Wah keren!\" di setiap respons — pakai ekspresi seperti itu HANYA jika konteksnya memang lucu/santai).\nJangan campur gaya formal dan informal dalam satu respons.\nDefaultnya: percakapan seperti asisten yang kompeten dan ramah, bukan hype-man.\nJangan gunakan filler exclamation di awal kalimat kecuali relevan dengan isi pesan user.",
	}

	if conf, err := b.dashRepo.GetAgentConfigByKey(ctx, "persona_name"); err == nil && conf != nil {
		json.Unmarshal([]byte(conf.ValueJSON), &p.Name)
	}
	if conf, err := b.dashRepo.GetAgentConfigByKey(ctx, "persona_description"); err == nil && conf != nil {
		json.Unmarshal([]byte(conf.ValueJSON), &p.Description)
	}
	if conf, err := b.dashRepo.GetAgentConfigByKey(ctx, "tone_preference"); err == nil && conf != nil {
		json.Unmarshal([]byte(conf.ValueJSON), &p.Tone)
	}
	if conf, err := b.dashRepo.GetAgentConfigByKey(ctx, "voice_guidelines"); err == nil && conf != nil {
		json.Unmarshal([]byte(conf.ValueJSON), &p.VoiceGuidelines)
	}

	return p
}

func (b *builder) loadPromptTemplate(ctx context.Context) string {
	template := ""
	if conf, err := b.dashRepo.GetAgentConfigByKey(ctx, "system_prompt_template"); err == nil && conf != nil {
		json.Unmarshal([]byte(conf.ValueJSON), &template)
	}
	return template
}

func (b *builder) buildMetadataBlock(ctx context.Context, docID string) string {
	doc, err := b.repo.GetUserDocumentByID(ctx, docID)
	if err != nil || doc == nil {
		return ""
	}

	metadataJSON := doc.MetadataJSON
	if metadataJSON == "" || metadataJSON == "{}" {
		basicMeta := map[string]string{
			"title":  doc.Title,
			"author": doc.Author,
		}
		bBytes, _ := json.Marshal(basicMeta)
		metadataJSON = string(bBytes)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("--- DOCUMENT METADATA (AKTIF: document_id=%s) ---\n", doc.ID))
	sb.WriteString(metadataJSON)
	sb.WriteString("\n--- END METADATA ---\n")
	return sb.String()
}

func (b *builder) buildPreferencesBlock(ctx context.Context, userID string) string {
	prefs, err := b.prefManager.GetAll(ctx, userID)
	if err != nil || len(prefs) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("[User Preferences]\n")
	for _, p := range prefs {
		if p.Source == "default" {
			continue // don't clutter prompt with defaults
		}
		sb.WriteString(fmt.Sprintf("- %s: %s (confidence: %s)\n", p.PrefKey, p.PrefValue, p.Confidence))
	}
	return sb.String()
}
