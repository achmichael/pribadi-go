package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/achmichael/pribadi-go/internal/delivery/whatsapp"
	"github.com/achmichael/pribadi-go/internal/repository"
	"github.com/achmichael/pribadi-go/internal/usecase/rag"
	"github.com/achmichael/pribadi-go/pkg/ollama"
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
	logger *zerolog.Logger,
) Orchestrator {
	return &orchestrator{waClient, transcription, extraction, ragIngest, ragRetrieve, ollamaClient, repo, logger}
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
		var err error
		var mime string

		dlStart := time.Now()
		if msg.MessageType == whatsapp.MessageTypeImage {
			imgMsg := msg.RawMessage.GetImageMessage()
			if imgMsg != nil {
				data, err = o.waClient.GetClient().Download(ctx, imgMsg)
				mime = imgMsg.GetMimetype()
			}
		} else {
			docMsg := msg.RawMessage.GetDocumentMessage()
			if docMsg != nil {
				data, err = o.waClient.GetClient().Download(ctx, docMsg)
				mime = docMsg.GetMimetype()
			}
		}
		if err != nil || data == nil {
			return fmt.Errorf("failed to download media: %v", err)
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
			
			metaPrompt := "Extract the Title and Author from the following document text. If not found, output 'Unknown'. Format exactly as:\nTitle: [Title]\nAuthor: [Author]\n\nDocument text:\n" + extractText
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
		Msg("[orchestrator] preprocessing complete, starting RAG retrieval")

	// ── 2. Retrieval ──────────────────────────────────────────────────
	retStart := time.Now()
	ctxInfo, err := o.ragRetrieve.Retrieve(ctx, userText)
	if err != nil {
		o.logger.Warn().Err(err).
			Str("msg_id", msg.ID).
			Dur("duration_ms", time.Since(retStart)).
			Msg("[orchestrator] RAG retrieval failed")
		// non-fatal: continue without context
		ctxInfo = rag.PromptContext{HasResults: false}
	}
	o.logger.Info().
		Str("msg_id", msg.ID).
		Bool("has_results", ctxInfo.HasResults).
		Int("context_len", len(ctxInfo.Context)).
		Dur("duration_ms", time.Since(retStart)).
		Msg("[orchestrator] RAG retrieval done")

	// ── 3. LLM Generation ─────────────────────────────────────────────
	systemPrompt := buildSystemPrompt(ctxInfo)

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
			Int("system_prompt_len", len(systemPrompt)).
			Int("user_text_len", len(userText)).
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

	return err
}

// buildSystemPrompt constructs system prompt with bounded RAG context.
func buildSystemPrompt(ctxInfo rag.PromptContext) string {
	base := "You are a helpful assistant. Answer concisely in the same language as the user's message. Keep answers under 150 words unless the user explicitly asks for detail."

	if !ctxInfo.HasResults {
		return base
	}

	ragCtx := ctxInfo.Context
	if len(ragCtx) > maxContextChars {
		ragCtx = ragCtx[:maxContextChars] + "\n[...context truncated...]"
	}

	var sb strings.Builder
	sb.WriteString(base)
	sb.WriteString("\n\nUse the following context from stored documents to help answer. ")
	sb.WriteString("If the context is not relevant, ignore it.\n\n")
	sb.WriteString("--- CONTEXT ---\n")
	sb.WriteString(ragCtx)
	sb.WriteString("\n--- END CONTEXT ---")

	if len(ctxInfo.Sources) > 0 {
		sb.WriteString("\n\nSources: ")
		sb.WriteString(strings.Join(ctxInfo.Sources, ", "))
	}

	return sb.String()
}
