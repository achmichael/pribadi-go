package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/achmichael/pribadi-go/internal/delivery/whatsapp"
	"github.com/achmichael/pribadi-go/internal/factmemory"
	"github.com/achmichael/pribadi-go/internal/identity"
	"github.com/achmichael/pribadi-go/internal/repository"
	"github.com/achmichael/pribadi-go/internal/usecase/rag"
	"github.com/achmichael/pribadi-go/pkg/ollama"
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
	memory        factmemory.MemoryManager
	resolver      *identity.Resolver
	refResolver   ReferenceResolver
	logger        *zerolog.Logger
}

func NewOrchestrator(
	waClient *whatsapp.Client,
	transcription TranscriptionService,
	extraction ExtractionService,
	ragIngest rag.IngestionService,
	ragRetrieve rag.RetrievalService,
	ollamaClient *ollama.OllamaClient,
	repo repository.Repository,
	memory factmemory.MemoryManager,
	resolver *identity.Resolver,
	refResolver ReferenceResolver,
	logger *zerolog.Logger,
) Orchestrator {
	return &orchestrator{
		waClient:      waClient,
		transcription: transcription,
		extraction:    extraction,
		ragIngest:     ragIngest,
		ragRetrieve:   ragRetrieve,
		ollama:        ollamaClient,
		repo:          repo,
		memory:        memory,
		resolver:      resolver,
		refResolver:   refResolver,
		logger:        logger,
	}
}

// maxContextChars caps how much RAG context goes into the system prompt.
const maxContextChars = 4000

func (o *orchestrator) Handle(ctx context.Context, msg whatsapp.IncomingMessage) error {
	handleStart := time.Now()
	o.logger.Info().
		Str("msg_id", msg.ID).
		Str("sender", msg.SenderJID).
		Str("type", string(msg.MessageType)).
		Msg("[orchestrator] handle start")

	// ── 0. Resolve platform identity → internal UserID ─────────────
	userID, err := o.resolver.ResolveUserID(ctx, "whatsapp", msg.SenderJID)
	if err != nil {
		o.logger.Error().Err(err).
			Str("sender", msg.SenderJID).
			Msg("[orchestrator] user resolution failed")
		return fmt.Errorf("resolve user: %w", err)
	}

	var userText string

	// ── 1. Preprocessing ──────────────────────────────────────────────
	switch msg.MessageType {
	case whatsapp.MessageTypeVoiceNote:
		// 1a. Download audio
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

		// 1b. Transcribe
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
		// 1a. Download media
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

		// 1b. Extract text
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

		// Debug log for extracted text (truncated to 1000 chars to avoid overwhelming the console)
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
			// Document: ingest into RAG, reply with confirmation only.

			// === EXTRACT METADATA VIA LLM ===
			metaStart := time.Now()
			metadata := map[string]string{"source_file": msg.ID}

			extractText := text
			if len(extractText) > 2000 {
				extractText = extractText[:2000]
			}

			metaPrompt := `You are the DOCUMENT TITLE EXTRACTION AND IDENTIFICATION AGENT.
You are responsible for identifying the true document title and author.
A document may contain file names, document types (e.g., LAPORAN, SKRIPSI), institution names, or report labels. These are NOT the document title.
The actual title is the primary intellectual work being presented.

PRIORITY ORDER for Title:
1. Title extracted from cover page.
2. Largest semantic title on first page.

Format exactly as:
Title: [Actual Title or Unknown]
Author: [Author or Unknown]

Document text:
` + extractText
			metaMessages := []ollama.ChatMessage{
				{Role: "user", Content: metaPrompt},
			}

			metaReply, err := o.ollama.Chat(ctx, metaMessages)
			if err == nil {
				for _, line := range strings.Split(metaReply, "\n") {
					line = strings.TrimSpace(line)
					lowerLine := strings.ToLower(line)
					if strings.HasPrefix(lowerLine, "title:") {
						metadata["Title"] = strings.TrimSpace(line[6:])
					} else if strings.HasPrefix(lowerLine, "author:") {
						metadata["Author"] = strings.TrimSpace(line[7:])
					}
				}
			} else {
				o.logger.Warn().Err(err).Msg("[orchestrator] metadata extraction failed, using default")
			}

			o.logger.Info().
				Str("msg_id", msg.ID).
				Dur("duration_ms", time.Since(metaStart)).
				Msg("[orchestrator] LLM metadata extraction done")

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
			
			docID := uuid.New().String()
			errDoc := o.repo.InsertUserDocument(ctx, repository.InsertUserDocumentParams{
				ID:            docID,
				UserID:        userID,
				PlatformMsgID: msg.ID,
				FileName:      fileName,
				Title:         metadata["Title"],
				Author:        metadata["Author"],
			})
			if errDoc != nil {
				o.logger.Warn().Err(errDoc).Msg("[orchestrator] failed to save document metadata to sqlite")
			} else {
				o.logger.Info().Str("doc_id", docID).Msg("[orchestrator] saved document metadata to sqlite")
			}

			sendStart := time.Now()
			summary := fmt.Sprintf("✅ Dokumen berhasil diproses.\n📄 %d bagian disimpan ke memori.\n\nKamu bisa langsung bertanya tentang isi dokumen ini.", count)
			err = o.waClient.SendText(ctx, msg.SenderJID, summary)
			o.logger.Info().
				Str("msg_id", msg.ID).
				Dur("duration_ms", time.Since(sendStart)).
				Dur("total_ms", time.Since(handleStart)).
				Msg("[orchestrator] WA send done (document)")
			return err
		}

	default:
		userText = msg.TextContent
	}

	o.logger.Info().
		Str("msg_id", msg.ID).
		Int("user_text_len", len(userText)).
		Dur("preprocess_ms", time.Since(handleStart)).
		Msg("[orchestrator] preprocessing complete, starting retrieval + memory")

	sessionID := "default"

	// ── 1.5 Reference Resolution ────────────────────────────────────
	ragQuery := userText
	targetDocID := ""
	if msg.MessageType == whatsapp.MessageTypeText || msg.MessageType == whatsapp.MessageTypeVoiceNote {
		ragQuery, targetDocID = o.refResolver.ResolveQuery(ctx, userID, userText, sessionID)
		if targetDocID != "" {
			// If target document is identified, we could eventually filter Qdrant by source_file=targetDocID
			// But for now, we just use the rewritten query.
			_ = targetDocID
		}
	}

	// ── 2. Retrieval (RAG docs + fact memory in parallel) ──────────
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

	go func() {
		retStart := time.Now()
		ctxInfo, err := o.ragRetrieve.Retrieve(ctx, ragQuery)
		o.logger.Info().
			Str("msg_id", msg.ID).
			Bool("has_results", ctxInfo.HasResults).
			Int("context_len", len(ctxInfo.Context)).
			Dur("ms", time.Since(retStart)).
			Msg("[orchestrator] RAG retrieval done")
		ragCh <- ragResult{ctx: ctxInfo, err: err}
	}()

	go func() {
		memStart := time.Now()
		memCtx, err := o.memory.PrefetchRelevant(ctx, userID, userText)
		o.logger.Info().
			Str("msg_id", msg.ID).
			Int("memory_len", len(memCtx)).
			Dur("ms", time.Since(memStart)).
			Msg("[orchestrator] memory prefetch done")
		memCh <- memResult{context: memCtx, err: err}
	}()

	ragRes := <-ragCh
	memRes := <-memCh

	ctxInfo := ragRes.ctx
	if ragRes.err != nil {
		o.logger.Warn().Err(ragRes.err).Msg("[orchestrator] RAG retrieval failed")
		ctxInfo = rag.PromptContext{HasResults: false}
	}

	memoryContext := ""
	if memRes.err != nil {
		o.logger.Warn().Err(memRes.err).Msg("[orchestrator] memory prefetch failed")
	} else {
		memoryContext = memRes.context
	}

	// ── 3. LLM Generation ─────────────────────────────────────────────
	systemPrompt := buildSystemPrompt(ctxInfo, memoryContext)

	messages := []ollama.ChatMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userText},
	}

	o.logger.Info().
		Str("msg_id", msg.ID).
		Int("system_prompt_len", len(systemPrompt)).
		Int("user_text_len", len(userText)).
		Msg("[orchestrator] sending to Ollama")

	llmStart := time.Now()
	reply, err := o.ollama.Chat(ctx, messages)
	llmDur := time.Since(llmStart)
	if err != nil {
		o.logger.Error().Err(err).
			Str("msg_id", msg.ID).
			Dur("duration_ms", llmDur).
			Msg("[orchestrator] Ollama chat FAILED")
		return err
	}
	o.logger.Info().
		Str("msg_id", msg.ID).
		Int("reply_len", len(reply)).
		Dur("duration_ms", llmDur).
		Msg("[orchestrator] Ollama chat done")

	// ── 4. Send reply ─────────────────────────────────────────────────
	sendStart := time.Now()
	err = o.waClient.SendText(ctx, msg.SenderJID, reply)
	o.logger.Info().
		Str("msg_id", msg.ID).
		Dur("send_ms", time.Since(sendStart)).
		Dur("total_ms", time.Since(handleStart)).
		Msg("[orchestrator] handle complete")

	// ── 5. Async: persist conversation + extract facts ────────────────
	// Store messages in messages_v2
	sessionID = "default" // TODO: session rotation based on idle time
	o.repo.InsertMessageV2(ctx, repository.InsertMessageV2Params{
		UserID:        userID,
		Platform:      "whatsapp",
		PlatformMsgID: msg.ID,
		SessionID:     sessionID,
		Role:          "user",
		Content:       userText,
		TokenCount:    estimateTokens(userText),
	})
	o.repo.InsertMessageV2(ctx, repository.InsertMessageV2Params{
		UserID:    userID,
		Platform:  "whatsapp",
		SessionID: sessionID,
		Role:      "assistant",
		Content:   reply,
		TokenCount: estimateTokens(reply),
	})

	// Fire async fact extraction
	o.memory.SyncAsync(userID, userText, reply)

	return err
}

// estimateTokens rough word-based token count.
func estimateTokens(s string) int {
	words := 0
	for range strings.Fields(s) {
		words++
	}
	return int(float64(words) * 1.3)
}

// buildSystemPrompt constructs system prompt with bounded RAG context + memory.
func buildSystemPrompt(ctxInfo rag.PromptContext, memoryContext string) string {
	base := `You are a helpful assistant. Answer concisely in the same language as the user's message. Keep answers under 150 words unless the user explicitly asks for detail.

When a user asks for a document's title (e.g., "What is the title?"):
- Return ONLY the actual document title.
- Do NOT prepend phrases like "File name", "Document type", "Institution name", or "Report type".
- Example of CORRECT response: "Analisis Pola Pembelian Konsumen Menggunakan Market Basket Analysis"
- Example of INCORRECT response: "File laporan_project_akhir.pdf memiliki judul..."`

	var sb strings.Builder
	sb.WriteString(base)

	// Memory context (personal facts from previous conversations)
	if memoryContext != "" {
		sb.WriteString("\n\n")
		sb.WriteString(memoryContext)
	}

	// RAG document context
	if ctxInfo.HasResults {
		ragCtx := ctxInfo.Context
		if len(ragCtx) > maxContextChars {
			ragCtx = ragCtx[:maxContextChars] + "\n[...context truncated...]"
		}

		sb.WriteString("\n\nUse the following context from stored documents to help answer. ")
		sb.WriteString("If the context is not relevant, ignore it.\n\n")
		sb.WriteString("--- CONTEXT ---\n")
		sb.WriteString(ragCtx)
		sb.WriteString("\n--- END CONTEXT ---")

		if len(ctxInfo.Sources) > 0 {
			sb.WriteString("\n\nSources: ")
			sb.WriteString(strings.Join(ctxInfo.Sources, ", "))
		}
	}

	return sb.String()
}
