package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/achmichael/pribadi-go/internal/budget"
	"github.com/achmichael/pribadi-go/internal/classifier"
	contextpkg "github.com/achmichael/pribadi-go/internal/context"
	"github.com/achmichael/pribadi-go/internal/continuation"
	"github.com/achmichael/pribadi-go/internal/conversation"
	"github.com/achmichael/pribadi-go/internal/domain"
	"github.com/achmichael/pribadi-go/internal/factmemory"
	"github.com/achmichael/pribadi-go/internal/prompt"
	"github.com/achmichael/pribadi-go/internal/reasoning"
	"github.com/achmichael/pribadi-go/internal/repository"
	"github.com/achmichael/pribadi-go/internal/response"
	"github.com/achmichael/pribadi-go/internal/usecase/rag"
	"github.com/achmichael/pribadi-go/pkg/llm"
	"github.com/achmichael/pribadi-go/pkg/utils"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type canonicalIntent struct {
	label     string
	embedding []float32
}

type DocMeta struct {
	DocumentType    string `json:"document_type"`
	Title           string `json:"title"`
	Authors         string `json:"authors"`
	Institution     string `json:"institution"`
	PublicationYear string `json:"publication_year"`
	DOI             string `json:"doi"`
	Keywords        string `json:"keywords"`
	Abstract        string `json:"abstract"`
	Supervisor      string `json:"supervisor"`
	Advisor         string `json:"advisor"`
}

type RouteDecision struct {
	Intent     string
	Confidence float32
}

type RetrievalStrategy int

const (
	StrategySkipAll RetrievalStrategy = iota
	StrategyAgentic
	StrategyPrefetch
)

const maxContextChars = 15000
const maxToolIterations = 3
const highConfidenceThreshold = 0.90

type coreOrchestrator struct {
	ragIngest     rag.IngestionService
	ragRetrieve   rag.RetrievalService
	llmRouter        llm.Router
	repo          repository.Repository
	dashboardRepo repository.DashboardRepository
	memory        factmemory.MemoryManager
	embedder      *utils.Embedder

	stateManager      conversation.StateManager
	prefManager       conversation.PreferenceManager
	intentClassifier  classifier.IntentClassifier
	contextBuilder    contextpkg.Builder
	promptBuilder     *prompt.Builder
	promptComposer    prompt.Composer
	verifier          reasoning.Verifier
	reflector         reasoning.Reflector
	responseProcessor response.Processor
	interactionLogger response.InteractionLogger
	contextualizer    continuation.Contextualizer
	logger            *zerolog.Logger

	canonicalIntents []canonicalIntent
}

func NewCoreOrchestrator(
	ragIngest rag.IngestionService,
	ragRetrieve rag.RetrievalService,
	llmRouter llm.Router,
	repo repository.Repository,
	dashboardRepo repository.DashboardRepository,
	memory factmemory.MemoryManager,
	embedder *utils.Embedder,
	stateManager conversation.StateManager,
	prefManager conversation.PreferenceManager,
	intentClassifier classifier.IntentClassifier,
	contextBuilder contextpkg.Builder,
	promptBuilder *prompt.Builder,
	promptComposer prompt.Composer,
	verifier reasoning.Verifier,
	reflector reasoning.Reflector,
	responseProcessor response.Processor,
	interactionLogger response.InteractionLogger,
	contextualizer continuation.Contextualizer,
	logger *zerolog.Logger,
) CoreOrchestrator {
	o := &coreOrchestrator{
		ragIngest:         ragIngest,
		ragRetrieve:       ragRetrieve,
		llmRouter:        llmRouter,
		repo:              repo,
		dashboardRepo:     dashboardRepo,
		memory:            memory,
		embedder:          embedder,
		stateManager:      stateManager,
		prefManager:       prefManager,
		intentClassifier:  intentClassifier,
		contextBuilder:    contextBuilder,
		promptBuilder:     promptBuilder,
		promptComposer:    promptComposer,
		verifier:          verifier,
		reflector:         reflector,
		responseProcessor: responseProcessor,
		interactionLogger: interactionLogger,
		contextualizer:    contextualizer,
		logger:            logger,
	}

	go o.warmCanonicalIntents()

	return o
}

func (o *coreOrchestrator) warmCanonicalIntents() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	intents := []struct {
		label string
		text  string
	}{
		{"chitchat", "halo apa kabar selamat pagi hi hello hey"},
		{"greeting", "salam kenal perkenalan"},
		{"thanks", "terima kasih makasih thanks thank you"},
		{"preference", "jawab dalam bahasa panggil aku pakai format"},
		{"ask_personal", "siapa nama saya dimana saya tinggal apa pekerjaan saya"},
		{"ask_document", "apa isi dokumen ringkas file jelaskan paper"},
		{"ask_document_metadata", "siapa penulis dokumen ini dari kampus mana judulnya apa tahun berapa dibuat apa institusi penulisnya author document"},
	}

	for _, intent := range intents {
		emb, err := o.embedder.Embed(ctx, "search_query: "+intent.text)
		if err != nil {
			o.logger.Warn().Err(err).Str("intent", intent.label).Msg("[orchestrator] canonical intent embedding failed")
			continue
		}
		o.canonicalIntents = append(o.canonicalIntents, canonicalIntent{
			label:     intent.label,
			embedding: emb,
		})
	}

	o.logger.Info().Int("count", len(o.canonicalIntents)).Msg("[orchestrator] canonical intents warmed")
}

func (o *coreOrchestrator) routeIntent(ctx context.Context, userText string) string {
	if len(o.canonicalIntents) == 0 {
		return ""
	}

	emb, err := o.embedder.Embed(ctx, "search_query: "+userText)
	if err != nil {
		return ""
	}

	bestScore := float32(-1)
	bestLabel := ""
	for _, ci := range o.canonicalIntents {
		score := cosineSimilarity(emb, ci.embedding)
		if score > bestScore {
			bestScore = score
			bestLabel = ci.label
		}
	}

	o.logger.Info().
		Str("best_intent", bestLabel).
		Float32("score", bestScore).
		Msg("[orchestrator] embedding router result")

	if bestScore >= 0.55 { // Lower threshold a bit since sometimes embeddings don't perfectly align
		return bestLabel
	}
	return ""
}

func cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}
	denom := math.Sqrt(normA) * math.Sqrt(normB)
	if denom == 0 {
		return 0
	}
	return float32(dot / denom)
}

func isDefinitelyNoRetrievalIntent(intent string) bool {
	noRetrievalIntents := map[string]bool{
		"greeting":  true,
		"thanks":    true,
		"chitchat": true,
	}
	return noRetrievalIntents[intent]
}

func isDefinitelyNeedsRetrievalIntent(intent string) bool {
	needsRetrievalIntents := map[string]bool{
		"ask_document":          true,
		"document_query":        true,
		"ask_document_metadata": true,
	}
	return needsRetrievalIntents[intent]
}

func confidenceStringToFloat(confidence string) float32 {
	switch confidence {
	case "high":
		return 0.95
	case "medium":
		return 0.65
	case "low":
		return 0.35
	default:
		return 0.5
	}
}

func (o *coreOrchestrator) HandleMessage(ctx context.Context, msg *NormalizedInboundMessage, out OutboundAdapter) error {
	handleStart := time.Now()
	o.logger.Info().
		Str("msg_id", msg.MessageID).
		Str("user_id", msg.UserID).
		Str("platform", msg.PlatformID).
		Str("session", msg.SessionID).
		Msg("[orchestrator] handle start")

	userText := msg.Text

	if strings.TrimSpace(strings.ToLower(userText)) == "/reset" {
		return o.handleReset(ctx, msg, out)
	}

	state, err := o.stateManager.Load(ctx, msg.UserID, msg.SessionID)
	if err != nil {
		o.logger.Warn().Err(err).Msg("[orchestrator] state load failed, continuing with empty state")
	}
	var stateData *domain.StateData
	if state != nil {
		stateData = state
	}

	heuristicStart := time.Now()
	classResult := o.intentClassifier.HeuristicOnly(classifier.ClassifyParams{
		UserText: userText,
	})
	o.logger.Info().
		Dur("ms", time.Since(heuristicStart)).
		Msg("[orchestrator] heuristic classify done")

	if classResult != nil && classResult.Intent == classifier.IntentSetPreference {
		intercepted, ackMsg, intErr := o.handlePreferenceIntercept(ctx, msg.UserID, msg.SessionID, userText, classResult, stateData)
		if intErr != nil {
			o.logger.Warn().Err(intErr).Msg("[orchestrator] interceptor error")
		}
		if intercepted {
			out.Deliver(ctx, msg.SessionID, ResponseChunk{
				Type: ChunkDone,
				Text: ackMsg,
			})
			o.persistAndLog(msg.UserID, msg.SessionID, msg.MessageID, userText, ackMsg, classResult, handleStart, nil)
			return nil
		}
	}

	contextualizeStart := time.Now()
	historyMsgs, err := o.repo.ListMessagesByUserSession(ctx, msg.UserID, msg.SessionID, 6)
	if err != nil {
		o.logger.Warn().Err(err).Msg("[orchestrator] failed to load history for contextualization")
	}

	var history []continuation.HistoryMessage
	for _, hm := range historyMsgs {
		history = append(history, continuation.HistoryMessage{
			Role:    hm.Role,
			Content: hm.Content,
		})
	}

	analysisResult, err := o.contextualizer.Analyze(ctx, userText, history)
	if err != nil {
		o.logger.Warn().Err(err).Msg("[orchestrator] contextualization failed, treating as standalone")
		analysisResult = &continuation.AnalysisResult{
			Type:           continuation.TypeStandalone,
			RewrittenQuery: userText,
		}
	}

	o.logger.Info().
		Str("analysis_type", string(analysisResult.Type)).
		Str("rewritten_query", analysisResult.RewrittenQuery).
		Dur("ms", time.Since(contextualizeStart)).
		Msg("[orchestrator] contextualization done")

	isContinuation := analysisResult.Type == continuation.TypeContinue
	if !isContinuation && analysisResult.RewrittenQuery != "" {
		userText = analysisResult.RewrittenQuery
	}

	routerStart := time.Now()
	routedIntent := o.routeIntent(ctx, userText)
	o.logger.Info().
		Str("routed_intent", routedIntent).
		Dur("ms", time.Since(routerStart)).
		Msg("[orchestrator] embedding router done")

	out.Deliver(ctx, msg.SessionID, ResponseChunk{
		Type:  ChunkStageEvent,
		Stage: "routing",
	})

	targetDocID := ""
	for _, att := range msg.Attachments {
		if att.Type == AttachmentTypeDocument && att.ExtractedText != "" {
			return o.handleDocumentUpload(ctx, msg, att.ExtractedText, handleStart, out)
		}
	}

	if o.shouldResolveReferences(userText) {
		resolvedQuery, resolvedDocID := o.resolveReferences(ctx, msg.UserID, userText, msg.SessionID)
		if resolvedQuery != "" {
			userText = resolvedQuery
			if resolvedDocID != "" {
				targetDocID = resolvedDocID
			}
		}
	}

	decision := RouteDecision{
		Intent:     routedIntent,
		Confidence: 0.65,
	}
	if classResult != nil && classResult.Confidence != "" {
		decision.Confidence = confidenceStringToFloat(classResult.Confidence)
	}

	strategy := o.decideRetrievalStrategy(decision, targetDocID)

	// Short-circuit for document metadata questions
	if decision.Intent == "ask_document_metadata" {
		if targetDocID == "" {
			// If no targetDocID is explicitly resolved, try to get the latest active document
			docs, err := o.repo.GetLatestUserDocuments(ctx, msg.UserID, 1)
			if err == nil && len(docs) > 0 {
				targetDocID = docs[0].ID
			}
		}

		if targetDocID != "" {
			doc, err := o.repo.GetUserDocumentByID(ctx, targetDocID)
			if err == nil && doc.ID != "" {
				var meta map[string]interface{}
				json.Unmarshal([]byte(doc.MetadataJSON), &meta)
				
				// Instead of a rigid short-circuit, we generate a synthetic chunk
				// that strictly contains ONLY the metadata properties, and pass it to the LLM 
				// as forced context. 
				
				metadataContext := fmt.Sprintf("Berikut adalah data metadata resmi dari dokumen %s:\n", doc.FileName)
				if doc.Title != "" && doc.Title != "Unknown" {
					metadataContext += fmt.Sprintf("Judul: %s\n", doc.Title)
				}
				if doc.Author != "" && doc.Author != "Unknown" {
					metadataContext += fmt.Sprintf("Penulis: %s\n", doc.Author)
				}
				if meta != nil {
					if inst, ok := meta["institution"].(string); ok && inst != "" {
						metadataContext += fmt.Sprintf("Institusi: %s\n", inst)
					}
					if year, ok := meta["publication_year"].(string); ok && year != "" {
						metadataContext += fmt.Sprintf("Tahun Terbit: %s\n", year)
					}
					if dt, ok := meta["document_type"].(string); ok && dt != "" {
						metadataContext += fmt.Sprintf("Jenis Dokumen: %s\n", dt)
					}
				}

				// We force the context builder to use THIS exact text as RAG context instead of searching vector DB.
				// By overriding `skipRAG` and setting `targetDocID = "METADATA_INJECT"`, we can hijack the context step.
				// However, a simpler way is to just inject it directly into the user text or handle it here via LLM call.

				// To ensure the tone matches, we let the LLM generate the response based ONLY on this injected context.
				o.logger.Info().
					Str("doc_id", targetDocID).
					Msg("[orchestrator] routing document metadata request to LLM with constrained context")

				// Format a prompt forcing the LLM to just answer the user's specific question based on the metadata block.
				systemPrompt := `Anda adalah asisten cerdas yang menjawab pertanyaan pengguna HANYA berdasarkan konteks metadata berikut. Jangan mengarang informasi di luar konteks ini. Jawab dengan ramah dan natural (tone: casual), jangan terlihat kaku seperti robot. Jawab langsung pada inti pertanyaannya saja.`
				
				messages := []llm.ChatMessage{
					{Role: "system", Content: systemPrompt + "\n\nKONTEKS METADATA:\n" + metadataContext},
					{Role: "user", Content: userText},
				}

				llmStart := time.Now()
				out.Deliver(ctx, msg.SessionID, ResponseChunk{
					Type:  ChunkStageEvent,
					Stage: "generating",
				})

				// Let's use default client for metadata operations
				activeLLM := o.llmRouter.GetDefaultClient()
				streamCh, streamErr := activeLLM.ChatStream(ctx, messages, nil)
				var rawResponse string

				if streamErr != nil {
					result, err := activeLLM.ChatWithTools(ctx, messages, "")
					if err != nil {
						o.logger.Error().Err(err).Msg("[orchestrator] constrained LLM generation failed")
						return err
					}
					rawResponse = result.Content
				} else {
					var fullContent strings.Builder
					for chunk := range streamCh {
						if chunk.Err != nil {
							break
						}
						fullContent.WriteString(chunk.Content)
						if chunk.Content != "" {
							out.Deliver(ctx, msg.SessionID, ResponseChunk{
								Type: ChunkToken,
								Text: chunk.Content,
							})
						}
					}
					rawResponse = fullContent.String()
				}

				llmDur := time.Since(llmStart)

				// Wrap up response processor stuff manually since we skipped the main flow
				finalRes, _ := o.responseProcessor.Process(ctx, response.ProcessParams{
					UserID:         msg.UserID,
					SessionID:      msg.SessionID,
					UserMessage:    userText,
					RawResponse:    rawResponse,
					Classification: classResult,
					Plan:           nil,
					ComposedPrompt: nil, 
					Verification:   nil,
					RAGSources:     []string{doc.FileName}, // cite the file
					State:          stateData,
				})

				finalRes.Metadata.TotalDurationMs = time.Since(handleStart).Milliseconds()
				finalRes.Metadata.LLMInferenceMs = llmDur.Milliseconds()
				finalRes.Metadata.ConfidenceScore = 1.0 // It's deterministically sourced

				out.Deliver(ctx, msg.SessionID, ResponseChunk{
					Type: ChunkDone,
					Text: finalRes.Text,
				})

				o.persistAndLog(msg.UserID, msg.SessionID, msg.MessageID, userText, finalRes.Text, classResult, handleStart, nil)
				return nil
			}
		}
	}

	o.logger.Info().
		Str("intent", decision.Intent).
		Float32("confidence", decision.Confidence).
		Str("strategy", strategyName(strategy)).
		Msg("[orchestrator] retrieval strategy decided")

	contextStart := time.Now()
	var skipRAG, skipMemory bool
	var toolSchemas []llm.Tool

	var activeLLM llm.Client
	if isContinuation {
		skipRAG = true
		skipMemory = true
		toolSchemas = nil
		
		activeLLM = o.llmRouter.GetDefaultClient()
		o.logger.Info().Msg("[orchestrator] CONTINUE mode: skipping RAG/memory, preparing verbatim history")
	} else {
		activeLLM, _ = o.llmRouter.GetClient(o.llmRouter.Route(decision.Intent, !skipRAG, !skipMemory))
		
	switch strategy {
		case StrategySkipAll:
			skipRAG = true
			skipMemory = true
			toolSchemas = nil

		case StrategyAgentic:
			skipRAG = true
			skipMemory = true
			toolSchemas = activeLLM.Schemas()

		case StrategyPrefetch:
			skipRAG = false
			skipMemory = false
			toolSchemas = activeLLM.Schemas()
		}
	}

	out.Deliver(ctx, msg.SessionID, ResponseChunk{
		Type:  ChunkStageEvent,
		Stage: "building_context",
	})

	sc, err := o.contextBuilder.Build(ctx, contextpkg.BuildParams{
		UserID:       msg.UserID,
		SessionID:    msg.SessionID,
		UserText:     userText,
		RAGQuery:     userText,
		TargetDocID:  targetDocID,
		HistoryLimit: 10,
		SkipRAG:      skipRAG,
		SkipMemory:   skipMemory,
	})
	if err != nil {
		return fmt.Errorf("context builder: %w", err)
	}
	o.logger.Info().
		Dur("ms", time.Since(contextStart)).
		Msg("[orchestrator] context build done")

	var convHistory []prompt.HistoryMessage
	var lastAssistantWasTruncated bool
	var lastAssistantContent string
	
	if historyMsgs, err := o.repo.ListMessagesByUserSession(ctx, msg.UserID, msg.SessionID, 10); err == nil {
		for _, m := range historyMsgs {
			convHistory = append(convHistory, prompt.HistoryMessage{
				Role:    m.Role,
				Content: m.Content,
			})
			
			if m.Role == "assistant" && m.Metadata != "" {
				var meta map[string]interface{}
				if json.Unmarshal([]byte(m.Metadata), &meta) == nil {
					if truncated, ok := meta["truncated"].(bool); ok && truncated {
						lastAssistantWasTruncated = true
						lastAssistantContent = m.Content
					}
				}
			}
		}
	}

	var messages []llm.ChatMessage
	
	if isContinuation {
		messages = []llm.ChatMessage{
			{Role: "system", Content: "Kamu adalah asisten AI yang membantu user. Lanjutkan jawabanmu sebelumnya persis dari titik terakhir. Jangan mengulang bagian yang sudah kamu sampaikan sebelumnya."},
		}
		
		for _, h := range convHistory {
			messages = append(messages, llm.ChatMessage{
				Role:    h.Role,
				Content: h.Content,
			})
		}
		
		if lastAssistantWasTruncated && lastAssistantContent != "" {
			messages = append(messages, llm.ChatMessage{
				Role:    "assistant",
				Content: lastAssistantContent,
			})
			o.logger.Info().Msg("[orchestrator] using assistant prefill for truncated continuation")
		}
		
		messages = append(messages, llm.ChatMessage{
			Role:    "user",
			Content: userText,
		})
	} else {
		composed, err := o.promptComposer.Compose(ctx, prompt.ComposeParams{
			UserText:            userText,
			Classification:      classResult,
			Plan:                nil,
			SessionContext:      sc,
			ConversationHistory: convHistory,
		})
		if err != nil {
			return fmt.Errorf("prompt composer: %w", err)
		}

		messages = []llm.ChatMessage{
			{Role: "system", Content: composed.SystemPrompt},
			{Role: "user", Content: composed.UserPrompt},
		}
	}

	o.logger.Info().
		Str("msg_id", msg.MessageID).
		Bool("is_continuation", isContinuation).
		Int("message_count", len(messages)).
		Msg("[orchestrator] sending unified call to Ollama")

	llmCallCount := 0
	llmStart := time.Now()
	var lastDoneReason string

	out.Deliver(ctx, msg.SessionID, ResponseChunk{
		Type:  ChunkStageEvent,
		Stage: "generating",
	})

	streamCh, streamErr := activeLLM.ChatStream(ctx, messages, toolSchemas)
	var rawResponse string

	if streamErr != nil {
		o.logger.Warn().Err(streamErr).Msg("[orchestrator] stream failed, falling back to non-stream")
		result, err := activeLLM.ChatWithTools(ctx, messages, "")
		if err != nil {
			llmDur := time.Since(llmStart)
			o.logger.Error().Err(err).Dur("ms", llmDur).Msg("[orchestrator] Ollama generation failed")
			o.interactionLogger.LogError(ctx, msg.UserID, msg.SessionID, userText, err.Error(), time.Since(handleStart).Milliseconds())
			return err
		}
		llmCallCount++
		rawResponse = result.Content
		lastDoneReason = result.DoneReason

		if len(result.ToolCalls) > 0 {
			rawResponse, llmCallCount, err = o.handleToolCalls(ctx, messages, result.ToolCalls, msg.UserID, userText, targetDocID, llmCallCount, out, activeLLM)
			if err != nil {
				o.interactionLogger.LogError(ctx, msg.UserID, msg.SessionID, userText, err.Error(), time.Since(handleStart).Milliseconds())
				return err
			}
		}
	} else {
		llmCallCount++
		var fullContent strings.Builder
		var accumulatedToolCalls []llm.ToolCall

		for chunk := range streamCh {
			if chunk.Err != nil {
				o.logger.Warn().Err(chunk.Err).Msg("[orchestrator] stream chunk error")
				break
			}
			fullContent.WriteString(chunk.Content)
			if len(chunk.ToolCalls) > 0 {
				accumulatedToolCalls = append(accumulatedToolCalls, chunk.ToolCalls...)
			}
			if chunk.DoneReason != "" {
				lastDoneReason = chunk.DoneReason
			}

			if chunk.Content != "" {
				out.Deliver(ctx, msg.SessionID, ResponseChunk{
					Type: ChunkToken,
					Text: chunk.Content,
				})
			}
		}

		if len(accumulatedToolCalls) > 0 {
			rawResponse, llmCallCount, err = o.handleToolCalls(ctx, messages, accumulatedToolCalls, msg.UserID, userText, targetDocID, llmCallCount, out, activeLLM)
			if err != nil {
				o.interactionLogger.LogError(ctx, msg.UserID, msg.SessionID, userText, err.Error(), time.Since(handleStart).Milliseconds())
				return err
			}
		} else {
			rawResponse = fullContent.String()
		}

		o.logger.Info().
			Str("done_reason", lastDoneReason).
			Bool("truncated", lastDoneReason == "length").
			Msg("[orchestrator] stream done_reason captured")
	}

	llmDur := time.Since(llmStart)
	o.logger.Info().
		Str("msg_id", msg.MessageID).
		Int("llm_calls", llmCallCount).
		Dur("llm_ms", llmDur).
		Int("response_len", len(rawResponse)).
		Msg("[orchestrator] LLM generation complete")

	var composedPrompt *prompt.ComposedPrompt
	if !isContinuation {
		composedPrompt, _ = o.promptComposer.Compose(ctx, prompt.ComposeParams{
			UserText:            userText,
			Classification:      classResult,
			Plan:                nil,
			SessionContext:      sc,
			ConversationHistory: convHistory,
		})
	}

	finalRes, err := o.responseProcessor.Process(ctx, response.ProcessParams{
		UserID:         msg.UserID,
		SessionID:      msg.SessionID,
		UserMessage:    userText,
		RawResponse:    rawResponse,
		Classification: classResult,
		Plan:           nil,
		ComposedPrompt: composedPrompt,
		Verification:   nil,
		RAGSources:     sc.RAGSources,
		State:          stateData,
	})
	if err != nil {
		return fmt.Errorf("response processor: %w", err)
	}

	finalRes.Metadata.TotalDurationMs = time.Since(handleStart).Milliseconds()
	finalRes.Metadata.LLMInferenceMs = llmDur.Milliseconds()

	out.Deliver(ctx, msg.SessionID, ResponseChunk{
		Type: ChunkDone,
		Text: finalRes.Text,
	})

	o.logger.Info().
		Str("msg_id", msg.MessageID).
		Dur("total_ms", time.Since(handleStart)).
		Int("llm_calls", llmCallCount).
		Msg("[orchestrator] response sent")

	bgCtx := context.Background()

	platform := strings.Split(msg.PlatformID, ":")[0]
	_, errInsertUser := o.repo.InsertMessageV2(bgCtx, repository.InsertMessageV2Params{
		UserID:        msg.UserID,
		Platform:      platform,
		PlatformMsgID: msg.MessageID,
		SessionID:     msg.SessionID,
		Role:          "user",
		Content:       userText,
		TokenCount:    estimateTokens(userText),
		Metadata:      "{}",
	})
	if errInsertUser != nil {
		o.logger.Error().Err(errInsertUser).Msg("[orchestrator] failed to insert user message to history")
	}

	assistantMetadata := map[string]interface{}{}
	if lastDoneReason == "length" {
		assistantMetadata["truncated"] = true
		assistantMetadata["done_reason"] = lastDoneReason
	}
	assistantMetadataJSON, _ := json.Marshal(assistantMetadata)

	_, errInsertAsst := o.repo.InsertMessageV2(bgCtx, repository.InsertMessageV2Params{
		UserID:     msg.UserID,
		Platform:   platform,
		SessionID:  msg.SessionID,
		Role:       "assistant",
		Content:    finalRes.Text,
		TokenCount: estimateTokens(finalRes.Text),
		Metadata:   string(assistantMetadataJSON),
	})
	if errInsertAsst != nil {
		o.logger.Error().Err(errInsertAsst).Msg("[orchestrator] failed to insert assistant message to history")
	}

	if finalRes.ShouldUpdate {
		o.responseProcessor.UpdateState(bgCtx, msg.UserID, msg.SessionID, classResult)
		o.responseProcessor.TriggerMemorySync(msg.UserID, userText, finalRes.Text)
	}

	responseSentAt := time.Now()
	go func() {
		verifyStart := time.Now()
		verification, _ := o.verifier.Verify(bgCtx, reasoning.VerifyParams{
			UserID:       msg.UserID,
			SessionID:    msg.SessionID,
			UserQuestion: userText,
			Response:     rawResponse,
			State:        stateData,
			RAGContext:   sc.RAGContext,
			RAGSources:   sc.RAGSources,
		})
		o.logger.Info().
			Dur("verify_ms", time.Since(verifyStart)).
			Dur("after_send_ms", time.Since(responseSentAt)).
			Msg("[orchestrator] async verification done (after response sent)")

		if verification != nil && !verification.IsValid {
			o.logger.Warn().
				Float32("confidence", verification.ConfidenceScore).
				Str("hallucination_risk", verification.HallucinationRisk).
				Int("issues", len(verification.Issues)).
				Msg("[orchestrator] async verification flagged response as invalid")
		}

		reflectStart := time.Now()
		reflection, _ := o.reflector.Reflect(bgCtx, reasoning.ReflectionParams{
			UserQuestion: userText,
			Response:     finalRes.Text,
			RAGUsed:      finalRes.Metadata.RAGUsed,
			MemoryUsed:   finalRes.Metadata.MemoryUsed,
			HistoryUsed:  finalRes.Metadata.HistoryUsed,
		})
		o.logger.Info().
			Dur("reflect_ms", time.Since(reflectStart)).
			Dur("after_send_ms", time.Since(responseSentAt)).
			Msg("[orchestrator] async reflection done (after response sent)")

		metaMap := map[string]interface{}{
			"llm_ms":    finalRes.Metadata.LLMInferenceMs,
			"llm_calls": llmCallCount,
		}
		if classResult != nil {
			metaMap["intent"] = classResult.Intent
			metaMap["class"] = classResult.MessageClass
		}
		if verification != nil {
			metaMap["confidence_score"] = verification.ConfidenceScore
			metaMap["hallucination_risk"] = verification.HallucinationRisk
			metaMap["is_valid"] = verification.IsValid
		}
		if reflection != nil {
			metaMap["reflection_confidence"] = reflection.ConfidenceLevel
		}

		o.interactionLogger.Log(bgCtx, response.InteractionLog{
			UserID:      msg.UserID,
			SessionID:   msg.SessionID,
			Timestamp:   time.Now(),
			UserMessage: userText,
			Response:    finalRes.Text,
			Metadata:    metaMap,
			Duration:    finalRes.Metadata.TotalDurationMs,
			Success:     true,
		})
	}()

	return nil
}

func (o *coreOrchestrator) handleToolCalls(ctx context.Context, messages []llm.ChatMessage, toolCalls []llm.ToolCall, userID, ragQuery, targetDocID string, callCount int, out OutboundAdapter, activeLLM llm.Client) (string, int, error) {
	currentMessages := make([]llm.ChatMessage, len(messages))
	copy(currentMessages, messages)

	for iteration := 0; iteration < maxToolIterations && len(toolCalls) > 0; iteration++ {
		currentMessages = append(currentMessages, llm.ChatMessage{
			Role:      "assistant",
			ToolCalls: toolCalls,
		})

		for _, tc := range toolCalls {
			out.Deliver(ctx, "", ResponseChunk{
				Type:     ChunkToolCall,
				ToolName: tc.Function.Name,
			})

			toolResult := o.executeToolCall(ctx, tc, userID, ragQuery, targetDocID)
			currentMessages = append(currentMessages, llm.ChatMessage{
				Role:    "tool",
				Content: toolResult,
			})
		}

		result, err := activeLLM.ChatWithTools(ctx, currentMessages, "")
		callCount++
		if err != nil {
			return "", callCount, fmt.Errorf("tool follow-up call: %w", err)
		}

		if len(result.ToolCalls) == 0 {
			return result.Content, callCount, nil
		}
		toolCalls = result.ToolCalls
	}

	lastResult, err := activeLLM.ChatWithToolsDirect(ctx, currentMessages, "", nil)
	callCount++
	if err != nil {
		return "", callCount, fmt.Errorf("final call after tool cap: %w", err)
	}
	return lastResult.Content, callCount, nil
}

func (o *coreOrchestrator) executeToolCall(ctx context.Context, tc llm.ToolCall, userID, ragQuery, targetDocID string) string {
	o.logger.Info().
		Str("tool", tc.Function.Name).
		Msg("[orchestrator] executing tool call")

	switch tc.Function.Name {
	case "search_documents":
		query := ragQuery
		if q, ok := tc.Function.Arguments["query"].(string); ok && q != "" {
			query = q
		}
		ctxInfo, err := o.ragRetrieve.Retrieve(ctx, query, targetDocID)
		if err != nil {
			return fmt.Sprintf("Error searching documents: %v", err)
		}
		if !ctxInfo.HasResults {
			return "No relevant documents found."
		}
		result := ctxInfo.Context
		if len(result) > maxContextChars {
			result = result[:maxContextChars] + "\n[...truncated...]"
		}
		return result

	case "recall_memory", "search_memory":
		query := ""
		if q, ok := tc.Function.Arguments["query"].(string); ok {
			query = q
		}
		if query == "" {
			query = ragQuery
		}
		memCtx, err := o.memory.PrefetchRelevant(ctx, userID, query)
		if err != nil {
			return fmt.Sprintf("Error searching memory: %v", err)
		}
		if memCtx == "" {
			return "No relevant personal facts found."
		}
		return memCtx

	default:
		return fmt.Sprintf("Unknown tool: %s", tc.Function.Name)
	}
}

func (o *coreOrchestrator) handleDocumentUpload(ctx context.Context, msg *NormalizedInboundMessage, text string, handleStart time.Time, out OutboundAdapter) error {
	if strings.TrimSpace(text) == "" {
		out.Deliver(ctx, msg.SessionID, ResponseChunk{
			Type: ChunkDone,
			Text: "❌ Gagal membaca isi dokumen. Pastikan file bukan gambar yang tidak bisa terbaca teksnya.",
		})
		return nil
	}

	out.Deliver(ctx, msg.SessionID, ResponseChunk{
		Type: ChunkStageEvent,
		Stage: "Sedang membaca dan memproses dokumen, tunggu sebentar ya ⏳",
	})

	metaStart := time.Now()
	metadata := map[string]string{"source_file": msg.MessageID}

	fileName := "unknown"
	for _, att := range msg.Attachments {
		if att.Type == AttachmentTypeDocument && att.FileName != "" {
			fileName = att.FileName
			break
		}
	}
	metadata["filename"] = fileName

	extractText := text
	if len(extractText) > 2000 {
		extractText = extractText[:2000]
	}
	extractText = strings.Join(strings.Fields(extractText), " ")

	metaPrompt := `ANDA ADALAH: DOCUMENT METADATA EXTRACTION AGENT

TUGAS ANDA:
Ekstrak metadata terstruktur dari potongan teks dokumen di bawah ini. Teks ini adalah hasil OCR/ekstraksi otomatis dari PDF dan SERING MENGANDUNG NOISE seperti: header jurnal, nomor volume/issue, kode artikel, running title dari artikel lain dalam volume yang sama, nomor halaman, watermark, atau metadata penerbit yang TIDAK BOLEH disalahartikan sebagai judul dokumen.

ATURAN WAJIB UNTUK MENENTUKAN "TITLE" (JUDUL):
1. Judul dokumen yang benar SELALU berada tepat SEBELUM daftar nama penulis (authors), bukan sebelum nama jurnal/header/volume.
2. JANGAN PERNAH mengambil teks yang terlihat seperti: nama jurnal, "Volume X Nomor Y", "JN-xxx"/kode artikel, ISSN, nama institusi penerbit, atau judul artikel lain yang mungkin muncul sebagai running header di bagian atas/bawah halaman.
3. Judul asli biasanya merupakan frasa nominal yang menjelaskan topik penelitian (metode, subjek, objek penelitian) — bukan kalimat administratif jurnal.
4. WAJIB lakukan cross-check internal sebelum finalisasi: setelah Anda mengekstrak "Title" dan "Abstract"/"Keywords", PERIKSA apakah keduanya secara tematik konsisten satu sama lain. Jika "Title" yang Anda temukan TIDAK relevan secara tema dengan "Abstract" atau "Keywords" yang Anda temukan di teks yang sama, maka "Title" tersebut SALAH — cari ulang kandidat judul lain di teks, atau jika benar-benar tidak ditemukan, kembalikan string kosong daripada memaksakan judul yang tidak konsisten.
5. Jika dalam satu potongan teks Anda menemukan LEBIH DARI SATU kandidat judul (misalnya satu di bagian atas sebagai header, satu lagi tepat sebelum nama penulis), PILIH yang posisinya TEPAT SEBELUM/BERDEKATAN dengan daftar nama penulis.
6. EKSTRAK SECARA VERBATIM (SAMA PERSIS): Salin teks judul asli kata demi kata. JANGAN PERNAH meringkas, menghilangkan kata, atau memodifikasi judul asli menjadi bentuk lain (contoh: teks asli yang panjang TIDAK BOLEH disingkat menjadi lebih pendek).

FORMAT OUTPUT:
Jawab HANYA dalam format JSON murni, tanpa markdown, tanpa basa-basi, tanpa penjelasan tambahan. Jika sebuah field tidak ditemukan dalam teks, isi dengan string kosong "" — JANGAN mengisi dengan tebakan atau placeholder.

{
  "document_type": "",
  "title": "",
  "authors": "",
  "institution": "",
  "publication_year": "",
  "doi": "",
  "keywords": "",
  "abstract": "",
  "supervisor": "",
  "advisor": ""
}

TEKS DOKUMEN UNTUK DIANALISIS:
` + extractText
	metaMessages := []llm.ChatMessage{
		{Role: "user", Content: metaPrompt},
	}

	var parsedMeta DocMeta
	var metaJSON string
	activeLLM := o.llmRouter.GetDefaultClient()
	metaReply, err := activeLLM.ChatJSON(ctx, metaMessages)
	if err == nil {
		cleanReply := strings.TrimSpace(metaReply)
		if strings.HasPrefix(cleanReply, "```json") {
			cleanReply = strings.TrimPrefix(cleanReply, "```json")
			cleanReply = strings.TrimSuffix(cleanReply, "```")
		} else if strings.HasPrefix(cleanReply, "```") {
			cleanReply = strings.TrimPrefix(cleanReply, "```")
			cleanReply = strings.TrimSuffix(cleanReply, "```")
		}
		cleanReply = strings.TrimSpace(cleanReply)

		if err := json.Unmarshal([]byte(cleanReply), &parsedMeta); err == nil {
			metadata["Title"] = parsedMeta.Title
			metadata["Author"] = parsedMeta.Authors
			metadata["DocumentType"] = parsedMeta.DocumentType
			metadata["Institution"] = parsedMeta.Institution
			metadata["PublicationYear"] = parsedMeta.PublicationYear

			metaBytes, _ := json.Marshal(parsedMeta)
			metaJSON = string(metaBytes)
		} else {
			o.logger.Warn().Err(err).Str("reply", cleanReply).Msg("[orchestrator] failed to parse JSON metadata")
		}
	} else {
		o.logger.Warn().Err(err).Msg("[orchestrator] metadata extraction failed")
	}

	if metadata["Title"] == "" {
		metadata["Title"] = "Unknown"
	}
	if metadata["Author"] == "" {
		metadata["Author"] = "Unknown"
	}
	if metaJSON == "" {
		metaJSON = "{}"
	}

	o.logger.Info().
		Str("msg_id", msg.MessageID).
		Dur("duration_ms", time.Since(metaStart)).
		Msg("[orchestrator] LLM metadata extraction done")

	docID := uuid.New().String()
	metadata["document_id"] = docID
	metadata["user_id"] = msg.UserID
	metadata["session_id"] = msg.SessionID
	metadata["uploaded_at"] = time.Now().UTC().Format(time.RFC3339)

	fileTokens := budget.EstimateTokens(text)
	calc := budget.NewCalculator(activeLLM.NumCtx(), activeLLM.NumPredict())
	var count int

	var metaDataObj map[string]interface{}
	json.Unmarshal([]byte(metaJSON), &metaDataObj)
	if metaDataObj == nil {
		metaDataObj = make(map[string]interface{})
	}
	metaDataObj["full_text"] = text
	metaDataBytes, _ := json.Marshal(metaDataObj)
	metaJSON = string(metaDataBytes)

	ingStart := time.Now()
	count, _ = o.ragIngest.IngestText(ctx, text, metadata)
	o.logger.Info().
		Str("msg_id", msg.MessageID).
		Int("chunks_stored", count).
		Int("text_len", len(text)).
		Int("file_tokens", fileTokens).
		Bool("fits_budget", calc.FitsInContext(fileTokens, "", nil)).
		Dur("duration_ms", time.Since(ingStart)).
		Msg("[orchestrator] RAG ingestion done (always chunk, full_text also saved)")

	errDoc := o.repo.InsertUserDocument(ctx, repository.InsertUserDocumentParams{
		ID:            docID,
		UserID:        msg.UserID,
		PlatformMsgID: msg.MessageID,
		FileName:      fileName,
		Title:         metadata["Title"],
		Author:        metadata["Author"],
		MetadataJSON:  metaJSON,
	})
	if errDoc != nil {
		o.logger.Warn().Err(errDoc).Msg("[orchestrator] failed to save document metadata to sqlite")
	} else {
		o.logger.Info().Str("doc_id", docID).Msg("[orchestrator] saved document metadata to sqlite")
	}

	summary := fmt.Sprintf("✅ Dokumen *%s* berhasil diproses.\n📄 %d bagian disimpan ke memori.\n\nKamu bisa langsung bertanya tentang isi dokumen ini.", fileName, count)
	out.Deliver(ctx, msg.SessionID, ResponseChunk{
		Type: ChunkDone,
		Text: summary,
	})

	o.logger.Info().
		Str("msg_id", msg.MessageID).
		Dur("total_ms", time.Since(handleStart)).
		Msg("[orchestrator] document upload done")

	platform := strings.Split(msg.PlatformID, ":")[0]
	_, errUser := o.repo.InsertMessageV2(context.Background(), repository.InsertMessageV2Params{
		UserID:        msg.UserID,
		Platform:      platform,
		PlatformMsgID: msg.MessageID,
		SessionID:     msg.SessionID,
		Role:          "user",
		Content:       "[Mengirim Dokumen: " + fileName + "]",
		TokenCount:    0,
	})
	if errUser != nil {
		o.logger.Error().Err(errUser).Msg("[orchestrator] failed to insert user doc upload to history")
	}

	_, errAsst := o.repo.InsertMessageV2(context.Background(), repository.InsertMessageV2Params{
		UserID:     msg.UserID,
		Platform:   platform,
		SessionID:  msg.SessionID,
		Role:       "assistant",
		Content:    summary,
		TokenCount: 0,
	})
	if errAsst != nil {
		o.logger.Error().Err(errAsst).Msg("[orchestrator] failed to insert assistant doc upload summary to history")
	}
	return nil
}

func (o *coreOrchestrator) handleReset(ctx context.Context, msg *NormalizedInboundMessage, out OutboundAdapter) error {
	o.logger.Info().Str("userID", msg.UserID).Str("sessionID", msg.SessionID).Msg("[Orchestrator] Explicit command received")

	err := o.repo.DeleteMessagesByUserSession(ctx, msg.UserID, msg.SessionID)
	if err != nil {
		o.logger.Warn().Err(err).Msg("[orchestrator] failed to delete messages from sqlite")
	}

	err = o.stateManager.DeleteState(ctx, msg.UserID, msg.SessionID)
	if err != nil {
		o.logger.Warn().Err(err).Msg("[orchestrator] failed to delete conversation state")
	}

	err = o.memory.PurgeMemory(ctx, msg.UserID)
	if err != nil {
		o.logger.Warn().Err(err).Msg("[orchestrator] failed to purge user fact")
	}

	err = o.ragRetrieve.PurgeRAGDocuments(ctx, msg.UserID)
	if err != nil {
		o.logger.Warn().Err(err).Msg("[orchestrator] failed to delete user documents from sqlite")
	}

	err = o.repo.DeleteAllUserDocuments(ctx, msg.UserID)
	if err != nil {
		o.logger.Warn().Err(err).Msg("[orchestrator] failed to purge user documents from qdrant")
	}

	out.Deliver(ctx, msg.SessionID, ResponseChunk{
		Type: ChunkDone,
		Text: "🧹 *Sesi Diatur Ulang*\nRiwayat percakapan dan memori sesi Anda telah berhasil dibersihkan.",
	})

	return nil
}

func (o *coreOrchestrator) handlePreferenceIntercept(ctx context.Context, userID, sessionID, userText string, classResult *classifier.Classification, stateData *domain.StateData) (bool, string, error) {
	if classResult == nil || len(classResult.ExtractedPrefs) == 0 {
		return false, "", nil
	}

	for _, pref := range classResult.ExtractedPrefs {
		_ = o.prefManager.SetExplicit(ctx, userID, pref.Key, pref.Value)
	}

	_ = o.stateManager.ApplyPreferences(ctx, userID, stateData)

	var ackParts []string
	for _, pref := range classResult.ExtractedPrefs {
		ackParts = append(ackParts, fmt.Sprintf("%s → %s", pref.Key, pref.Value))
	}
	ackMsg := "✅ Preferensi diperbarui: " + strings.Join(ackParts, ", ")

	return true, ackMsg, nil
}

func (o *coreOrchestrator) persistAndLog(userID, sessionID, msgID, userText, reply string, classResult *classifier.Classification, handleStart time.Time, sendErr error) {
	bgCtx := context.Background()

	platform := "unknown"
	o.repo.InsertMessageV2(bgCtx, repository.InsertMessageV2Params{
		UserID:        userID,
		Platform:      platform,
		PlatformMsgID: msgID,
		SessionID:     sessionID,
		Role:          "user",
		Content:       userText,
		TokenCount:    estimateTokens(userText),
	})
	o.repo.InsertMessageV2(bgCtx, repository.InsertMessageV2Params{
		UserID:     userID,
		Platform:   platform,
		SessionID:  sessionID,
		Role:       "assistant",
		Content:    reply,
		TokenCount: estimateTokens(reply),
	})

	o.responseProcessor.UpdateState(bgCtx, userID, sessionID, classResult)
	o.responseProcessor.TriggerMemorySync(userID, userText, reply)

	o.interactionLogger.Log(bgCtx, response.InteractionLog{
		UserID:      userID,
		SessionID:   sessionID,
		Timestamp:   time.Now(),
		UserMessage: userText,
		Response:    reply,
		Metadata:    map[string]interface{}{"intercepted": true},
		Duration:    time.Since(handleStart).Milliseconds(),
		Success:     sendErr == nil,
	})
}

func estimateTokens(s string) int {
	words := 0
	for range strings.Fields(s) {
		words++
	}
	return int(float64(words) * 1.3)
}

func (o *coreOrchestrator) shouldResolveReferences(userText string) bool {
	lower := strings.ToLower(userText)
	keywords := []string{
		"ini", "itu", "tersebut", "tadi", "kemarin", "sebelumnya",
		"dokumen", "file", "paper", "artikel",
		"this", "that", "the document", "the file", "previous",
	}
	for _, kw := range keywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

func (o *coreOrchestrator) decideRetrievalStrategy(decision RouteDecision, targetDocID string) RetrievalStrategy {
	if targetDocID != "" {
		return StrategyPrefetch
	}

	if decision.Confidence >= highConfidenceThreshold && isDefinitelyNoRetrievalIntent(decision.Intent) {
		return StrategySkipAll
	}

	if decision.Confidence >= highConfidenceThreshold && isDefinitelyNeedsRetrievalIntent(decision.Intent) {
		return StrategyPrefetch
	}

	return StrategyAgentic
}

func strategyName(s RetrievalStrategy) string {
	switch s {
	case StrategySkipAll:
		return "fast_path_skip"
	case StrategyAgentic:
		return "agentic_decision"
	case StrategyPrefetch:
		return "fast_path_retrieve"
	default:
		return "unknown"
	}
}

func (o *coreOrchestrator) resolveReferences(ctx context.Context, userID, userText, sessionID string) (string, string) {
	docs, err := o.repo.GetLatestUserDocuments(ctx, userID, 3)
	if err != nil {
		o.logger.Warn().Err(err).Msg("[orchestrator] failed to fetch latest documents")
	}

	if len(docs) == 0 {
		return userText, ""
	}

	var docRegistry strings.Builder
	for i, doc := range docs {
		status := ""
		if i == 0 {
			status = " (ACTIVE DOCUMENT - NEWEST UPLOAD)"
		}
		docRegistry.WriteString(fmt.Sprintf("- document_id: %s%s\n  file_name: %s\n  title: %s\n  author: %s\n  uploaded_at: %s\n\n",
			doc.ID, status, doc.FileName, doc.Title, doc.Author, doc.CreatedAt.Format("2006-01-02 15:04:05")))
	}

	msgs, err := o.repo.ListMessagesByUserSession(ctx, userID, sessionID, 5)
	if err != nil {
		o.logger.Warn().Err(err).Msg("[orchestrator] failed to fetch conversation history")
	}

	var history strings.Builder
	for _, msg := range msgs {
		history.WriteString(fmt.Sprintf("%s: %s\n", strings.ToUpper(msg.Role), msg.Content))
	}

	prompt := fmt.Sprintf(`You are the DOCUMENT FOCUS AND ACTIVE DOCUMENT MANAGEMENT AGENT.
You are responsible for resolving document references and maintaining the currently active document.

You have access to:
* Uploaded document registry
* Conversation history

## ACTIVE DOCUMENT RULE
The "active document" is the most recently uploaded document. Only one document can be active at a time.
A newly uploaded document immediately becomes the active focus and overrides any previous active document.

## DOCUMENT REGISTRY
%s

## STRICT PROHIBITION & CONTEXT RESET RULE
Never answer questions about a previous document merely because it was discussed earlier.
Do not reuse metadata from older documents. The newest uploaded document always replaces the old focus, UNLESS the user explicitly names a previous document or says "kembali ke dokumen sebelumnya".

## RECENCY PRIORITY RULE
If the user says things like:
"dokumen ini", "file tersebut", "judulnya", "penulisnya", "apa judulnya", "isi bab 3", "ringkas dokumen ini", dll.

Resolve using Priority:
1. active_document (most recently uploaded document)
2. explicitly named document
3. clarification request

## CONVERSATION HISTORY (Last 5 messages)
%s

## TASK
Step 1: Analyze whether the user is referring to a document indirectly or directly.
Step 2: Resolve the reference using the strict ACTIVE DOCUMENT RULE.
	Step 3: Rewrite the query by embedding the file_name or title instead of the raw document_id. Example: User: "Siapa penulisnya?", Rewritten: "Siapa penulis dari dokumen 'Proposal Skripsi.pdf'?"

IMPORTANT: Return ONLY a valid JSON object with the following structure, and no other text:
{
  "rewritten_query": "the rewritten query containing the context, or the original query if no reference is found",
  "target_document_id": "the resolved document ID, or empty string if none"
}

	Current User Query: %s
	`, docRegistry.String(), history.String(), userText)

	messages := []llm.ChatMessage{
		{Role: "user", Content: prompt},
	}

	activeLLM := o.llmRouter.GetDefaultClient()
	reply, err := activeLLM.ChatJSON(ctx, messages)
	if err != nil {
		o.logger.Error().Err(err).Msg("[orchestrator] reference resolution LLM call failed")
		return userText, ""
	}

	reply = strings.TrimSpace(reply)
	if reply == "" || reply == "{}" {
		return userText, ""
	}

	var result struct {
		RewrittenQuery   string `json:"rewritten_query"`
		TargetDocumentID string `json:"target_document_id"`
	}
	if err := json.Unmarshal([]byte(reply), &result); err != nil {
		o.logger.Warn().Err(err).Str("reply", reply).Msg("[orchestrator] failed to parse resolution JSON")
		return userText, ""
	}

	if result.RewrittenQuery != "" {
		o.logger.Info().
			Str("original", userText).
			Str("rewritten", result.RewrittenQuery).
			Str("target_doc", result.TargetDocumentID).
			Msg("[orchestrator] query resolved")
		return result.RewrittenQuery, result.TargetDocumentID
	}

	return userText, ""
}
