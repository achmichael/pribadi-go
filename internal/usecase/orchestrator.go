package usecase

import (
	"context"
	"fmt"
	"strings"

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
// 5 chunks × ~400 tokens × 4 chars/token ≈ 8000. We use 6000 for safety.
const maxContextChars = 6000

func (o *orchestrator) Handle(ctx context.Context, msg whatsapp.IncomingMessage) error {
	var userText string
	var skipLLM bool // when true, send a fixed reply without calling Ollama

	// 1. Preprocessing
	switch msg.MessageType {
	case whatsapp.MessageTypeVoiceNote:
		audioMsg := msg.RawMessage.GetAudioMessage()
		if audioMsg == nil {
			return fmt.Errorf("no audio message found")
		}
		data, err := o.waClient.GetClient().Download(ctx, audioMsg)
		if err != nil {
			return err
		}
		text, err := o.transcription.Transcribe(ctx, data)
		if err != nil {
			return err
		}
		userText = text

	case whatsapp.MessageTypeImage, whatsapp.MessageTypeDocument:
		var data []byte
		var err error
		var mime string
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

		// Use effective mime from message if IncomingMessage field is empty
		if mime == "" {
			mime = msg.MediaMimetype
		}

		text, err := o.extraction.ExtractText(ctx, data, mime)
		if err != nil {
			return err
		}

		if msg.MessageType == whatsapp.MessageTypeImage {
			// Image: Florence already produced rich description.
			// Feed directly to Ollama for conversational response.
			userText = text
			if msg.Caption != "" {
				userText = fmt.Sprintf("%s\n\nUser caption: %s", text, msg.Caption)
			}
		} else {
			// Document: ingest into RAG, reply with confirmation only.
			// DO NOT send full doc text to LLM — that causes timeouts.
			count, _ := o.ragIngest.IngestText(ctx, text, map[string]string{"source_file": msg.ID})

			o.logger.Info().
				Int("chunks_stored", count).
				Int("text_len", len(text)).
				Str("msg_id", msg.ID).
				Msg("Document ingested into RAG")

			// Send confirmation directly — no LLM call needed
			summary := fmt.Sprintf("✅ Dokumen berhasil diproses.\n📄 %d bagian disimpan ke memori.\n\nKamu bisa langsung bertanya tentang isi dokumen ini.", count)
			return o.waClient.SendText(ctx, msg.SenderJID, summary)
		}

	default:
		userText = msg.TextContent
	}

	if skipLLM {
		return nil
	}

	// 2. Retrieval & Generation
	ctxInfo, _ := o.ragRetrieve.Retrieve(ctx, userText)

	systemPrompt := buildSystemPrompt(ctxInfo)

	messages := []ollama.ChatMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userText},
	}

	reply, err := o.ollama.Chat(ctx, messages)
	if err != nil {
		o.logger.Error().Err(err).
			Int("user_text_len", len(userText)).
			Int("system_prompt_len", len(systemPrompt)).
			Msg("Ollama chat failed")
		return err
	}

	// 3. Response
	return o.waClient.SendText(ctx, msg.SenderJID, reply)
}

// buildSystemPrompt constructs system prompt with bounded RAG context.
func buildSystemPrompt(ctxInfo rag.PromptContext) string {
	base := "You are a helpful assistant. Answer in the same language as the user's message."

	if !ctxInfo.HasResults {
		return base
	}

	ragCtx := ctxInfo.Context
	// Truncate RAG context if too long to prevent prompt blowup
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
