package usecase

import (
	"context"
	"fmt"

	"github.com/achmichael/pribadi-go/internal/delivery/whatsapp"
	"github.com/achmichael/pribadi-go/internal/repository"
	"github.com/achmichael/pribadi-go/internal/usecase/rag"
	"github.com/achmichael/pribadi-go/pkg/ollama"
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
}

func NewOrchestrator(
	waClient *whatsapp.Client,
	transcription TranscriptionService,
	extraction ExtractionService,
	ragIngest rag.IngestionService,
	ragRetrieve rag.RetrievalService,
	ollama *ollama.OllamaClient,
	repo repository.Repository,
) Orchestrator {
	return &orchestrator{waClient, transcription, extraction, ragIngest, ragRetrieve, ollama, repo}
}

func (o *orchestrator) Handle(ctx context.Context, msg whatsapp.IncomingMessage) error {
	var userText string

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
			// Document: ingest into RAG then inform user
			_, _ = o.ragIngest.IngestText(ctx, text, map[string]string{"source_file": msg.ID})
			userText = fmt.Sprintf("Dokumen telah diproses dan disimpan. Isi:\n%s", text)
		}
	default:
		userText = msg.TextContent
	}

	// 2. Retrieval & Generation
	ctxInfo, _ := o.ragRetrieve.Retrieve(ctx, userText)
	
	systemPrompt := "You are a helpful assistant."
	if ctxInfo.HasResults {
		systemPrompt += " Use the following context: " + ctxInfo.Context
	}

	messages := []ollama.ChatMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userText},
	}

	reply, err := o.ollama.Chat(ctx, messages)
	if err != nil {
		return err
	}

	// 3. Response
	return o.waClient.SendText(ctx, msg.SenderJID, reply)
}
