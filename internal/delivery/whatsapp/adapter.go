package whatsapp

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/achmichael/pribadi-go/internal/identity"
	"github.com/achmichael/pribadi-go/internal/orchestrator"
	"github.com/achmichael/pribadi-go/internal/repository"
	"github.com/rs/zerolog"
)

type TranscriptionService interface {
	Transcribe(ctx context.Context, audioData []byte) (string, error)
}

type ExtractionService interface {
	ExtractText(ctx context.Context, fileBytes []byte, mimeType string) (string, error)
}

type whatsappInboundAdapter struct {
	waClient      *Client
	transcription TranscriptionService
	extraction    ExtractionService
	resolver      *identity.Resolver
	repo          repository.Repository
	logger        *zerolog.Logger
}

func NewInboundAdapter(
	waClient *Client,
	transcription TranscriptionService,
	extraction ExtractionService,
	resolver *identity.Resolver,
	repo repository.Repository,
	logger *zerolog.Logger,
) orchestrator.InboundAdapter {
	return &whatsappInboundAdapter{
		waClient:      waClient,
		transcription: transcription,
		extraction:    extraction,
		resolver:      resolver,
		repo:          repo,
		logger:        logger,
	}
}

func (a *whatsappInboundAdapter) Normalize(ctx context.Context, rawEvent any) (*orchestrator.NormalizedInboundMessage, error) {
	msg, ok := rawEvent.(IncomingMessage)
	if !ok {
		return nil, fmt.Errorf("expected IncomingMessage, got %T", rawEvent)
	}

	userID, err := a.resolver.ResolveUserID(ctx, "whatsapp", msg.SenderJID)
	if err != nil {
		return nil, fmt.Errorf("resolve user: %w", err)
	}

	sessionID, isNew, err := a.repo.GetOrCreateActiveSession(ctx, userID, "whatsapp", 120)
	if err != nil {
		return nil, fmt.Errorf("get/create session: %w", err)
	}
	if isNew {
		a.logger.Info().Str("user_id", userID).Str("session_id", sessionID).Msg("[whatsapp] created new session after idle timeout")
	}

	normalized := &orchestrator.NormalizedInboundMessage{
		UserID:      userID,
		PlatformID:  fmt.Sprintf("whatsapp:%s", msg.SenderJID),
		SessionID:   sessionID,
		MessageID:   msg.ID,
		Text:        "",
		Attachments: nil,
		Metadata:    make(map[string]any),
		Timestamp:   msg.Timestamp,
	}

	switch msg.MessageType {
	case MessageTypeText:
		normalized.Text = msg.TextContent

	case MessageTypeVoiceNote:
		audioMsg := msg.RawMessage.GetAudioMessage()
		if audioMsg == nil {
			return nil, fmt.Errorf("no audio message found")
		}

		dlStart := time.Now()
		data, err := a.waClient.GetClient().Download(ctx, audioMsg)
		if err != nil {
			return nil, err
		}
		a.logger.Info().
			Str("msg_id", msg.ID).
			Int("bytes", len(data)).
			Dur("duration_ms", time.Since(dlStart)).
			Msg("[whatsapp] media download done")

		txStart := time.Now()
		text, err := a.transcription.Transcribe(ctx, data)
		if err != nil {
			return nil, err
		}
		a.logger.Info().
			Str("msg_id", msg.ID).
			Int("text_len", len(text)).
			Dur("duration_ms", time.Since(txStart)).
			Msg("[whatsapp] transcription done")

		normalized.Text = text
		normalized.Attachments = append(normalized.Attachments, orchestrator.Attachment{
			Type:        orchestrator.AttachmentTypeVoice,
			Data:        data,
			MimeType:    audioMsg.GetMimetype(),
			Transcribed: text,
		})

	case MessageTypeImage, MessageTypeDocument:
		var data []byte
		var dlErr error
		var mime string
		var fileName string

		dlStart := time.Now()
		if msg.MessageType == MessageTypeImage {
			imgMsg := msg.RawMessage.GetImageMessage()
			if imgMsg != nil {
				data, dlErr = a.waClient.GetClient().Download(ctx, imgMsg)
				mime = imgMsg.GetMimetype()
			}
		} else {
			docMsg := msg.RawMessage.GetDocumentMessage()
			if docMsg != nil {
				data, dlErr = a.waClient.GetClient().Download(ctx, docMsg)
				mime = docMsg.GetMimetype()
				fileName = docMsg.GetFileName()
			}
		}
		if dlErr != nil || data == nil {
			return nil, fmt.Errorf("failed to download media: %v", dlErr)
		}
		a.logger.Info().
			Str("msg_id", msg.ID).
			Str("mime", mime).
			Int("bytes", len(data)).
			Dur("duration_ms", time.Since(dlStart)).
			Msg("[whatsapp] media download done")

		if mime == "" {
			mime = msg.MediaMimetype
		}

		exStart := time.Now()
		text, err := a.extraction.ExtractText(ctx, data, mime)
		if err != nil {
			return nil, err
		}
		a.logger.Info().
			Str("msg_id", msg.ID).
			Int("text_len", len(text)).
			Dur("duration_ms", time.Since(exStart)).
			Msg("[whatsapp] extraction done")

		if msg.MessageType == MessageTypeImage {
			normalized.Text = text
			if msg.Caption != "" {
				normalized.Text = fmt.Sprintf("%s\n\nUser caption: %s", text, msg.Caption)
			}
			normalized.Attachments = append(normalized.Attachments, orchestrator.Attachment{
				Type:          orchestrator.AttachmentTypeImage,
				Data:          data,
				MimeType:      mime,
				Caption:       msg.Caption,
				ExtractedText: text,
			})
		} else {
			normalized.Attachments = append(normalized.Attachments, orchestrator.Attachment{
				Type:          orchestrator.AttachmentTypeDocument,
				Data:          data,
				MimeType:      mime,
				FileName:      fileName,
				ExtractedText: text,
			})
		}

	default:
		return nil, fmt.Errorf("unsupported message type: %s", msg.MessageType)
	}

	return normalized, nil
}

type whatsappOutboundAdapter struct {
	waClient *Client
	logger   *zerolog.Logger
	buffer   strings.Builder
}

func NewOutboundAdapter(waClient *Client, logger *zerolog.Logger) orchestrator.OutboundAdapter {
	return &whatsappOutboundAdapter{
		waClient: waClient,
		logger:   logger,
	}
}

func (a *whatsappOutboundAdapter) Deliver(ctx context.Context, sessionID string, chunk orchestrator.ResponseChunk) error {
	switch chunk.Type {
	case orchestrator.ChunkStageEvent:
		return nil

	case orchestrator.ChunkToken:
		a.buffer.WriteString(chunk.Text)
		return nil

	case orchestrator.ChunkToolCall:
		return nil

	case orchestrator.ChunkDone:
		finalText := chunk.Text
		if a.buffer.Len() > 0 {
			finalText = a.buffer.String()
			a.buffer.Reset()
		}

		if finalText == "" {
			return nil
		}

		senderJID := ""
		if md, ok := ctx.Value("wa_sender_jid").(string); ok {
			senderJID = md
		}
		if senderJID == "" {
			return fmt.Errorf("wa_sender_jid not found in context")
		}

		return a.waClient.SendText(ctx, senderJID, finalText)

	case orchestrator.ChunkError:
		if chunk.Error != nil {
			a.logger.Error().Err(chunk.Error).Msg("[whatsapp] chunk error received")
		}
		return chunk.Error

	default:
		return nil
	}
}
