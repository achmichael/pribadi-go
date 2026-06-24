package usecase

import (
	"context"
	"encoding/json"
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

// DocMeta represents structured metadata extracted from documents
type DocMeta struct {
	DocumentType    string   `json:"document_type"`
	Title           string   `json:"title"`
	Authors         []string `json:"authors"`
	Institution     string   `json:"institution"`
	PublicationYear string   `json:"publication_year"`
	DOI             string   `json:"doi"`
	Keywords        []string `json:"keywords"`
	Abstract        string   `json:"abstract"`
	Supervisor      string   `json:"supervisor"`
	Advisor         string   `json:"advisor"`
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

			metaPrompt := `You are a document metadata extraction system.
Your task is to extract structured metadata from the first pages of a document.
The document may be a journal article, conference paper, thesis, dissertation, final project report, technical report, book, or research paper.

## IMPORTANT
Do not identify metadata based on file names. Do not use assumptions. Only use information explicitly present in the document.

## OUTPUT FORMAT
Return ONLY valid JSON.
{
  "document_type": "",
  "title": "",
  "authors": [],
  "institution": "",
  "publication_year": "",
  "doi": "",
  "keywords": [],
  "abstract": "",
  "supervisor": "",
  "advisor": ""
}
Never return markdown. Never return explanations. Only JSON.

## DOCUMENT TEXT
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
					metadata["Author"] = strings.Join(parsedMeta.Authors, ", ")
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
	metadataBlock := ""
	if msg.MessageType == whatsapp.MessageTypeText || msg.MessageType == whatsapp.MessageTypeVoiceNote {
		ragQuery, targetDocID = o.refResolver.ResolveQuery(ctx, userID, userText, sessionID)
		if targetDocID != "" {
			doc, err := o.repo.GetUserDocumentByID(ctx, targetDocID)
			if err == nil && doc != nil {
				var b strings.Builder
				b.WriteString("--- DOCUMENT METADATA ---\n")
				b.WriteString(fmt.Sprintf("File Name: %s\n", doc.FileName))
				
				var parsedMeta DocMeta
				hasRichMeta := false
				if doc.MetadataJSON != "" && doc.MetadataJSON != "{}" {
					if err := json.Unmarshal([]byte(doc.MetadataJSON), &parsedMeta); err == nil {
						hasRichMeta = true
						if parsedMeta.Title != "" { b.WriteString(fmt.Sprintf("Title: %s\n", parsedMeta.Title)) }
						if len(parsedMeta.Authors) > 0 { b.WriteString(fmt.Sprintf("Authors: %s\n", strings.Join(parsedMeta.Authors, ", "))) }
						if parsedMeta.DocumentType != "" { b.WriteString(fmt.Sprintf("Document Type: %s\n", parsedMeta.DocumentType)) }
						if parsedMeta.Institution != "" { b.WriteString(fmt.Sprintf("Institution: %s\n", parsedMeta.Institution)) }
						if parsedMeta.PublicationYear != "" { b.WriteString(fmt.Sprintf("Publication Year: %s\n", parsedMeta.PublicationYear)) }
						if parsedMeta.DOI != "" { b.WriteString(fmt.Sprintf("DOI: %s\n", parsedMeta.DOI)) }
						if len(parsedMeta.Keywords) > 0 { b.WriteString(fmt.Sprintf("Keywords: %s\n", strings.Join(parsedMeta.Keywords, ", "))) }
						if parsedMeta.Supervisor != "" { b.WriteString(fmt.Sprintf("Supervisor: %s\n", parsedMeta.Supervisor)) }
						if parsedMeta.Advisor != "" { b.WriteString(fmt.Sprintf("Advisor: %s\n", parsedMeta.Advisor)) }
					}
				}
				
				if !hasRichMeta {
					b.WriteString(fmt.Sprintf("Title: %s\n", doc.Title))
					b.WriteString(fmt.Sprintf("Author: %s\n", doc.Author))
				}
				b.WriteString("--- END METADATA ---\n")
				metadataBlock = b.String()
			}
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
		ctxInfo, err := o.ragRetrieve.Retrieve(ctx, ragQuery, targetDocID)
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
	systemPrompt := buildSystemPrompt(ctxInfo, memoryContext, metadataBlock)

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
		UserID:     userID,
		Platform:   "whatsapp",
		SessionID:  sessionID,
		Role:       "assistant",
		Content:    reply,
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
func buildSystemPrompt(ctxInfo rag.PromptContext, memoryContext, metadataBlock string) string {
	base := `You are a helpful assistant. Answer concisely in the same language as the user's message. Keep answers under 150 words unless the user explicitly asks for detail.

When a user asks for a document's title (e.g., "What is the title?"):
- Return ONLY the actual document title.
- Do NOT prepend phrases like "File name", "Document type", "Institution name", or "Report type".
- Example of CORRECT response: "Analisis Pola Pembelian Konsumen Menggunakan Market Basket Analysis"
- Example of INCORRECT response: "File laporan_project_akhir.pdf memiliki judul..."`

	var sb strings.Builder
	sb.WriteString(base)

	if metadataBlock != "" {
		sb.WriteString("\n\n")
		sb.WriteString(metadataBlock)
	}

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
