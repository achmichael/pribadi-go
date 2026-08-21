package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/achmichael/pribadi-go/internal/classifier"
	contextpkg "github.com/achmichael/pribadi-go/internal/context"
	"github.com/achmichael/pribadi-go/internal/conversation"
	"github.com/achmichael/pribadi-go/internal/delivery/whatsapp"
	"github.com/achmichael/pribadi-go/internal/domain"
	"github.com/achmichael/pribadi-go/internal/factmemory"
	"github.com/achmichael/pribadi-go/internal/identity"
	"github.com/achmichael/pribadi-go/internal/prompt"
	"github.com/achmichael/pribadi-go/internal/reasoning"
	"github.com/achmichael/pribadi-go/internal/repository"
	"github.com/achmichael/pribadi-go/internal/response"
	"github.com/achmichael/pribadi-go/internal/usecase/rag"
	"github.com/achmichael/pribadi-go/pkg/ollama"
	"github.com/achmichael/pribadi-go/pkg/utils"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type Orchestrator interface {
	Handle(ctx context.Context, msg whatsapp.IncomingMessage) error
}

type orchestrator struct {
	waClient      *whatsapp.Client
	transcription TranscriptionService
	extraction    ExtractionService
	ragIngest     rag.IngestionService
	ragRetrieve   rag.RetrievalService
	ollama        *ollama.OllamaClient
	repo          repository.Repository
	dashboardRepo repository.DashboardRepository
	memory        factmemory.MemoryManager
	resolver      *identity.Resolver
	refResolver   ReferenceResolver
	embedder      *utils.OllamaEmbedder

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
	logger            *zerolog.Logger

	canonicalIntents []canonicalIntent
}

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

const maxContextChars = 2000
const maxToolIterations = 3

func NewOrchestrator(
	waClient *whatsapp.Client,
	transcription TranscriptionService,
	extraction ExtractionService,
	ragIngest rag.IngestionService,
	ragRetrieve rag.RetrievalService,
	ollamaClient *ollama.OllamaClient,
	repo repository.Repository,
	dashboardRepo repository.DashboardRepository,
	memory factmemory.MemoryManager,
	resolver *identity.Resolver,
	refResolver ReferenceResolver,
	embedder *utils.OllamaEmbedder,
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
	logger *zerolog.Logger,
) Orchestrator {
	o := &orchestrator{
		waClient:          waClient,
		transcription:     transcription,
		extraction:        extraction,
		ragIngest:         ragIngest,
		ragRetrieve:       ragRetrieve,
		ollama:            ollamaClient,
		repo:              repo,
		dashboardRepo:     dashboardRepo,
		memory:            memory,
		resolver:          resolver,
		refResolver:       refResolver,
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
		logger:            logger,
	}

	go o.warmCanonicalIntents()

	return o
}

func (o *orchestrator) warmCanonicalIntents() {
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

func (o *orchestrator) routeIntent(ctx context.Context, userText string) string {
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

	if bestScore >= 0.65 {
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

func (o *orchestrator) Handle(ctx context.Context, msg whatsapp.IncomingMessage) error {
	handleStart := time.Now()
	o.logger.Info().
		Str("msg_id", msg.ID).
		Str("sender", msg.SenderJID).
		Str("type", string(msg.MessageType)).
		Msg("[orchestrator] handle start")

	userID, err := o.resolver.ResolveUserID(ctx, "whatsapp", msg.SenderJID)
	if err != nil {
		o.logger.Error().Err(err).
			Str("sender", msg.SenderJID).
			Msg("[orchestrator] user resolution failed")
		return fmt.Errorf("resolve user: %w", err)
	}

	var userText string
	sessionID := "default"

	switch msg.MessageType {
	case whatsapp.MessageTypeVoiceNote:
		audioMsg := msg.RawMessage.GetAudioMessage()
		if audioMsg == nil {
			return fmt.Errorf("no audio message found")
		}

		dlStart := time.Now()
		data, err := o.waClient.GetClient().Download(ctx, audioMsg)
		if err != nil {
			return err
		}
		o.logger.Info().
			Str("msg_id", msg.ID).
			Int("bytes", len(data)).
			Dur("duration_ms", time.Since(dlStart)).
			Msg("[orchestrator] media download done")

		txStart := time.Now()
		text, err := o.transcription.Transcribe(ctx, data)
		if err != nil {
			return err
		}
		o.logger.Info().
			Str("msg_id", msg.ID).
			Int("text_len", len(text)).
			Dur("duration_ms", time.Since(txStart)).
			Msg("[orchestrator] transcription done")

		userText = text

	case whatsapp.MessageTypeImage, whatsapp.MessageTypeDocument:
		var data []byte
		var dlErr error
		var mime string

		dlStart := time.Now()
		if msg.MessageType == whatsapp.MessageTypeImage {
			imgMsg := msg.RawMessage.GetImageMessage()
			if imgMsg != nil {
				data, dlErr = o.waClient.GetClient().Download(ctx, imgMsg)
				mime = imgMsg.GetMimetype()
			}
		} else {
			docMsg := msg.RawMessage.GetDocumentMessage()
			if docMsg != nil {
				data, dlErr = o.waClient.GetClient().Download(ctx, docMsg)
				mime = docMsg.GetMimetype()
			}
		}
		if dlErr != nil || data == nil {
			return fmt.Errorf("failed to download media: %v", dlErr)
		}
		o.logger.Info().
			Str("msg_id", msg.ID).
			Str("mime", mime).
			Int("bytes", len(data)).
			Dur("duration_ms", time.Since(dlStart)).
			Msg("[orchestrator] media download done")

		if mime == "" {
			mime = msg.MediaMimetype
		}

		exStart := time.Now()
		text, err := o.extraction.ExtractText(ctx, data, mime)
		if err != nil {
			return err
		}
		o.logger.Info().
			Str("msg_id", msg.ID).
			Int("text_len", len(text)).
			Dur("duration_ms", time.Since(exStart)).
			Msg("[orchestrator] extraction done")

		previewText := text
		if len(previewText) > 1000 {
			previewText = previewText[:1000] + "... (truncated)"
		}
		o.logger.Info().
			Str("msg_id", msg.ID).
			Str("extracted_text_preview", previewText).
			Msg("[orchestrator] extracted text content")

		if msg.MessageType == whatsapp.MessageTypeImage {
			userText = text
			if msg.Caption != "" {
				userText = fmt.Sprintf("%s\n\nUser caption: %s", text, msg.Caption)
			}
		} else {
			return o.handleDocumentUpload(ctx, msg, userID, sessionID, text, data, handleStart)
		}

	default:
		userText = msg.TextContent
	}

	o.logger.Info().
		Str("msg_id", msg.ID).
		Int("user_text_len", len(userText)).
		Dur("preprocess_ms", time.Since(handleStart)).
		Msg("[orchestrator] preprocessing complete, starting pipeline")

	if strings.TrimSpace(strings.ToLower(userText)) == "/reset" {
		return o.handleReset(ctx, msg, userID, sessionID)
	}

	state, err := o.stateManager.Load(ctx, userID, sessionID)
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
		intercepted, ackMsg, intErr := o.handlePreferenceIntercept(ctx, userID, sessionID, userText, classResult, stateData)
		if intErr != nil {
			o.logger.Warn().Err(intErr).Msg("[orchestrator] interceptor error")
		}
		if intercepted {
			sendErr := o.waClient.SendText(ctx, msg.SenderJID, ackMsg)
			o.persistAndLog(userID, sessionID, msg.ID, userText, ackMsg, classResult, handleStart, sendErr)
			return sendErr
		}
	}

	routerStart := time.Now()
	routedIntent := o.routeIntent(ctx, userText)
	o.logger.Info().
		Str("routed_intent", routedIntent).
		Dur("ms", time.Since(routerStart)).
		Msg("[orchestrator] embedding router done")

	ragQuery := userText
	targetDocID := ""
	if msg.MessageType == whatsapp.MessageTypeText || msg.MessageType == whatsapp.MessageTypeVoiceNote {
		ragQuery, targetDocID = o.refResolver.ResolveQuery(ctx, userID, userText, sessionID)
	}

	contextStart := time.Now()
	skipRAG := routedIntent == "chitchat" || routedIntent == "greeting" || routedIntent == "thanks"
	skipMemory := routedIntent == "greeting" || routedIntent == "thanks"

	sc, err := o.contextBuilder.Build(ctx, contextpkg.BuildParams{
		UserID:       userID,
		SessionID:    sessionID,
		UserText:     userText,
		RAGQuery:     ragQuery,
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
	if historyMsgs, err := o.repo.ListMessagesByUserSession(ctx, userID, sessionID, 10); err == nil {
		for _, m := range historyMsgs {
			convHistory = append(convHistory, prompt.HistoryMessage{
				Role:    m.Role,
				Content: m.Content,
			})
		}
	}

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

	messages := []ollama.ChatMessage{
		{Role: "system", Content: composed.SystemPrompt},
		{Role: "user", Content: composed.UserPrompt},
	}

	o.logger.Info().
		Str("msg_id", msg.ID).
		Int("sys_len", len(composed.SystemPrompt)).
		Int("user_len", len(composed.UserPrompt)).
		Msg("[orchestrator] sending unified call to Ollama")

	llmCallCount := 0
	llmStart := time.Now()

	streamCh, streamErr := o.ollama.ChatStream(ctx, messages, o.ollama.Schemas())
	var rawResponse string

	if streamErr != nil {
		o.logger.Warn().Err(streamErr).Msg("[orchestrator] stream failed, falling back to non-stream")
		result, err := o.ollama.ChatWithTools(ctx, messages, "")
		if err != nil {
			llmDur := time.Since(llmStart)
			o.logger.Error().Err(err).Dur("ms", llmDur).Msg("[orchestrator] Ollama generation failed")
			o.interactionLogger.LogError(ctx, userID, sessionID, userText, err.Error(), time.Since(handleStart).Milliseconds())
			return err
		}
		llmCallCount++
		rawResponse = result.Content

		if len(result.ToolCalls) > 0 {
			rawResponse, llmCallCount, err = o.handleToolCalls(ctx, messages, result.ToolCalls, userID, ragQuery, targetDocID, llmCallCount)
			if err != nil {
				o.interactionLogger.LogError(ctx, userID, sessionID, userText, err.Error(), time.Since(handleStart).Milliseconds())
				return err
			}
		}
	} else {
		llmCallCount++
		var fullContent strings.Builder
		// sentBuffer := ""
		// sentCount := 0
		const maxPartialSends = 5
		var accumulatedToolCalls []ollama.ToolCall

		for chunk := range streamCh {
			if chunk.Err != nil {
				o.logger.Warn().Err(chunk.Err).Msg("[orchestrator] stream chunk error")
				break
			}
			fullContent.WriteString(chunk.Content)
			if len(chunk.ToolCalls) > 0 {
				accumulatedToolCalls = append(accumulatedToolCalls, chunk.ToolCalls...)
			}

			// if !chunk.Done && sentCount < maxPartialSends {
			// 	sentBuffer += chunk.Content
			// 	if idx := findSentenceEnd(sentBuffer); idx > 0 && len(sentBuffer[:idx]) > 30 {
			// 		partial := strings.TrimSpace(sentBuffer[:idx])
			// 		if partial != "" {
			// 			// o.waClient.SendText(ctx, msg.SenderJID, partial)
			// 			sentCount++
			// 			o.logger.Debug().
			// 				Int("partial_num", sentCount).
			// 				Int("len", len(partial)).
			// 				Msg("[orchestrator] partial send")
			// 		}
			// 		sentBuffer = sentBuffer[idx:]
			// 	}
			// }
		}

		if len(accumulatedToolCalls) > 0 {
			rawResponse, llmCallCount, err = o.handleToolCalls(ctx, messages, accumulatedToolCalls, userID, ragQuery, targetDocID, llmCallCount)
			if err != nil {
				o.interactionLogger.LogError(ctx, userID, sessionID, userText, err.Error(), time.Since(handleStart).Milliseconds())
				return err
			}
		} else {
			rawResponse = fullContent.String()
		}
	}

	llmDur := time.Since(llmStart)
	o.logger.Info().
		Str("msg_id", msg.ID).
		Int("llm_calls", llmCallCount).
		Dur("llm_ms", llmDur).
		Int("response_len", len(rawResponse)).
		Msg("[orchestrator] LLM generation complete")

	finalRes, err := o.responseProcessor.Process(ctx, response.ProcessParams{
		UserID:         userID,
		SessionID:      sessionID,
		UserMessage:    userText,
		RawResponse:    rawResponse,
		Classification: classResult,
		Plan:           nil,
		ComposedPrompt: composed,
		Verification:   nil,
		RAGSources:     sc.RAGSources,
		State:          stateData,
	})
	if err != nil {
		return fmt.Errorf("response processor: %w", err)
	}

	finalRes.Metadata.TotalDurationMs = time.Since(handleStart).Milliseconds()
	finalRes.Metadata.LLMInferenceMs = llmDur.Milliseconds()

	sendStart := time.Now()
	sendTextErr := o.waClient.SendText(ctx, msg.SenderJID, finalRes.Text)
	o.logger.Info().
		Str("msg_id", msg.ID).
		Dur("send_ms", time.Since(sendStart)).
		Dur("total_ms", time.Since(handleStart)).
		Int("llm_calls", llmCallCount).
		Msg("[orchestrator] response sent")

	bgCtx := context.Background()

	o.repo.InsertMessageV2(bgCtx, repository.InsertMessageV2Params{
		UserID:        userID,
		Platform:      "whatsapp",
		PlatformMsgID: msg.ID,
		SessionID:     sessionID,
		Role:          "user",
		Content:       userText,
		TokenCount:    estimateTokens(userText),
	})
	o.repo.InsertMessageV2(bgCtx, repository.InsertMessageV2Params{
		UserID:     userID,
		Platform:   "whatsapp",
		SessionID:  sessionID,
		Role:       "assistant",
		Content:    finalRes.Text,
		TokenCount: estimateTokens(finalRes.Text),
	})

	if finalRes.ShouldUpdate {
		o.responseProcessor.UpdateState(bgCtx, userID, sessionID, classResult)
		o.responseProcessor.TriggerMemorySync(userID, userText, finalRes.Text)
	}

	responseSentAt := time.Now()
	go func() {
		verifyStart := time.Now()
		verification, _ := o.verifier.Verify(bgCtx, reasoning.VerifyParams{
			UserID:       userID,
			SessionID:    sessionID,
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
			UserID:      userID,
			SessionID:   sessionID,
			Timestamp:   time.Now(),
			UserMessage: userText,
			Response:    finalRes.Text,
			Metadata:    metaMap,
			Duration:    finalRes.Metadata.TotalDurationMs,
			Success:     sendTextErr == nil,
		})
	}()

	return sendTextErr
}

func (o *orchestrator) handleToolCalls(ctx context.Context, messages []ollama.ChatMessage, toolCalls []ollama.ToolCall, userID, ragQuery, targetDocID string, callCount int) (string, int, error) {
	currentMessages := make([]ollama.ChatMessage, len(messages))
	copy(currentMessages, messages)

	for iteration := 0; iteration < maxToolIterations && len(toolCalls) > 0; iteration++ {
		currentMessages = append(currentMessages, ollama.ChatMessage{
			Role:      "assistant",
			ToolCalls: toolCalls,
		})

		for _, tc := range toolCalls {
			toolResult := o.executeToolCall(ctx, tc, userID, ragQuery, targetDocID)
			currentMessages = append(currentMessages, ollama.ChatMessage{
				Role:    "tool",
				Content: toolResult,
			})
		}

		result, err := o.ollama.ChatWithTools(ctx, currentMessages, "")
		callCount++
		if err != nil {
			return "", callCount, fmt.Errorf("tool follow-up call: %w", err)
		}

		if len(result.ToolCalls) == 0 {
			return result.Content, callCount, nil
		}
		toolCalls = result.ToolCalls
	}

	lastResult, err := o.ollama.ChatWithToolsDirect(ctx, currentMessages, "", nil)
	callCount++
	if err != nil {
		return "", callCount, fmt.Errorf("final call after tool cap: %w", err)
	}
	return lastResult.Content, callCount, nil
}

func (o *orchestrator) executeToolCall(ctx context.Context, tc ollama.ToolCall, userID, ragQuery, targetDocID string) string {
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

	case "search_memory":
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

func (o *orchestrator) handleDocumentUpload(ctx context.Context, msg whatsapp.IncomingMessage, userID, sessionID, text string, _ []byte, handleStart time.Time) error {
	ackSendErr := o.waClient.SendText(ctx, msg.SenderJID, "Sedang membaca dan memproses dokumen, tunggu sebentar ya ⏳")
	if ackSendErr != nil {
		o.logger.Warn().Err(ackSendErr).Msg("[orchestrator] failed to send initial document ack")
	}

	metaStart := time.Now()
	metadata := map[string]string{"source_file": msg.ID}

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
	metaMessages := []ollama.ChatMessage{
		{Role: "user", Content: metaPrompt},
	}

	var parsedMeta DocMeta
	var metaJSON string
	metaReply, err := o.ollama.ChatJSON(ctx, metaMessages)
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
		Str("msg_id", msg.ID).
		Dur("duration_ms", time.Since(metaStart)).
		Msg("[orchestrator] LLM metadata extraction done")

	docID := uuid.New().String()
	metadata["document_id"] = docID
	metadata["user_id"] = userID

	ingStart := time.Now()
	count, _ := o.ragIngest.IngestText(ctx, text, metadata)
	o.logger.Info().
		Str("msg_id", msg.ID).
		Int("chunks_stored", count).
		Int("text_len", len(text)).
		Dur("duration_ms", time.Since(ingStart)).
		Msg("[orchestrator] RAG ingestion done")

	fileName := "unknown"
	if docMsg := msg.RawMessage.GetDocumentMessage(); docMsg != nil && docMsg.GetFileName() != "" {
		fileName = docMsg.GetFileName()
	}

	errDoc := o.repo.InsertUserDocument(ctx, repository.InsertUserDocumentParams{
		ID:            docID,
		UserID:        userID,
		PlatformMsgID: msg.ID,
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

	sendStart := time.Now()
	summary := fmt.Sprintf("✅ Dokumen *%s* berhasil diproses.\n📄 %d bagian disimpan ke memori.\n\nKamu bisa langsung bertanya tentang isi dokumen ini.", fileName, count)
	sendErr := o.waClient.SendText(ctx, msg.SenderJID, summary)
	o.logger.Info().
		Str("msg_id", msg.ID).
		Dur("duration_ms", time.Since(sendStart)).
		Dur("total_ms", time.Since(handleStart)).
		Msg("[orchestrator] WA send done (document)")

	o.repo.InsertMessageV2(context.Background(), repository.InsertMessageV2Params{
		UserID:        userID,
		Platform:      "whatsapp",
		PlatformMsgID: msg.ID,
		SessionID:     sessionID,
		Role:          "user",
		Content:       "[Mengirim Dokumen: " + fileName + "]",
		TokenCount:    0,
	})

	o.repo.InsertMessageV2(context.Background(), repository.InsertMessageV2Params{
		UserID:    userID,
		Platform:  "whatsapp",
		SessionID: sessionID,
		Role:      "assistant",
		Content:   summary,
		TokenCount: 0,
	})
	return sendErr
}

func (o *orchestrator) handleReset(ctx context.Context, msg whatsapp.IncomingMessage, userID, sessionID string) error {
	o.logger.Info().Str("userID", userID).Str("sessionID", sessionID).Msg("[Orchestrator] Explicit commmand received")

	err := o.repo.DeleteMessagesByUserSession(ctx, userID, sessionID)
	if err != nil {
		o.logger.Warn().Err(err).Msg("[orchestrator] failed to delete messages from sqlite")
	}

	err = o.stateManager.DeleteState(ctx, userID, sessionID)
	if err != nil {
		o.logger.Warn().Err(err).Msg("[orchestrator] failed to delete conversation state")
	}

	err = o.memory.PurgeMemory(ctx, userID)
	if err != nil {
		o.logger.Warn().Err(err).Msg("[orchestrator] failed to purge user fact")
	}

	err = o.ragRetrieve.PurgeRAGDocuments(ctx, userID)
	if err != nil {
		o.logger.Warn().Err(err).Msg("[orchestrator] failed to delete user documents from sqlite")
	}

	err = o.repo.DeleteAllUserDocuments(ctx, userID)
	if err != nil {
		o.logger.Warn().Err(err).Msg("[orchestrator] failed to purge user documents from qdrant")
	}

	sendErr := o.waClient.SendText(ctx, msg.SenderJID, "🧹 *Sesi Diatur Ulang*\nRiwayat percakapan dan memori sesi Anda telah berhasil dibersihkan.")
	if sendErr != nil {
		o.logger.Warn().Err(sendErr).Msg("[orchestrator] failed to send reset confirmation")
	}

	return sendErr
}

func (o *orchestrator) handlePreferenceIntercept(ctx context.Context, userID, sessionID, userText string, classResult *classifier.Classification, stateData *domain.StateData) (bool, string, error) {
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

func (o *orchestrator) persistAndLog(userID, sessionID, msgID, userText, reply string, classResult *classifier.Classification, handleStart time.Time, sendErr error) {
	bgCtx := context.Background()

	o.repo.InsertMessageV2(bgCtx, repository.InsertMessageV2Params{
		UserID:        userID,
		Platform:      "whatsapp",
		PlatformMsgID: msgID,
		SessionID:     sessionID,
		Role:          "user",
		Content:       userText,
		TokenCount:    estimateTokens(userText),
	})
	o.repo.InsertMessageV2(bgCtx, repository.InsertMessageV2Params{
		UserID:     userID,
		Platform:   "whatsapp",
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

func findSentenceEnd(s string) int {
	for _, sep := range []string{". ", ".\n", "! ", "!\n", "? ", "?\n"} {
		if idx := strings.Index(s, sep); idx > 0 {
			return idx + len(sep)
		}
	}
	if idx := strings.Index(s, "\n\n"); idx > 0 {
		return idx + 2
	}
	return -1
}

func estimateTokens(s string) int {
	words := 0
	for range strings.Fields(s) {
		words++
	}
	return int(float64(words) * 1.3)
}
