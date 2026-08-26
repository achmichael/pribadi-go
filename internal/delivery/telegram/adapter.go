package telegram

import (
	"context"
	"fmt"
	"io"
	"strings"
	"github.com/achmichael/pribadi-go/internal/identity"
	"github.com/achmichael/pribadi-go/internal/orchestrator"
	"github.com/achmichael/pribadi-go/internal/repository"
	"github.com/rs/zerolog"
	tele "gopkg.in/telebot.v3"
)

type TranscriptionService interface {
	Transcribe(ctx context.Context, audioData []byte) (string, error)
}

type ExtractionService interface {
	ExtractText(ctx context.Context, fileBytes []byte, mimeType string) (string, error)
}

// ─── Inbound Adapter ───────────────────────────────────────────────

type telegramInboundAdapter struct {
	bot           *tele.Bot
	transcription TranscriptionService
	extraction    ExtractionService
	resolver      *identity.Resolver
	repo          repository.Repository
	logger        *zerolog.Logger
}

func NewInboundAdapter(
	bot *tele.Bot,
	transcription TranscriptionService,
	extraction ExtractionService,
	resolver *identity.Resolver,
	repo repository.Repository,
	logger *zerolog.Logger,
) orchestrator.InboundAdapter {
	return &telegramInboundAdapter{
		bot:           bot,
		transcription: transcription,
		extraction:    extraction,
		resolver:      resolver,
		repo:          repo,
		logger:        logger,
	}
}

func (a *telegramInboundAdapter) Normalize(ctx context.Context, rawEvent any) (*orchestrator.NormalizedInboundMessage, error) {
	c, ok := rawEvent.(tele.Context)
	if !ok {
		return nil, fmt.Errorf("expected tele.Context, got %T", rawEvent)
	}

	msg := c.Message()
	if msg == nil {
		return nil, fmt.Errorf("message is nil")
	}

	platformUserID := fmt.Sprintf("%d", msg.Sender.ID)

	userID, err := a.resolver.ResolveUserID(ctx, "telegram", platformUserID)
	if err != nil {
		return nil, fmt.Errorf("resolve user: %w", err)
	}

	// 120 menit idle timeout seperti WhatsApp
	sessionID, isNew, err := a.repo.GetOrCreateActiveSession(ctx, userID, "telegram", 120)
	if err != nil {
		return nil, fmt.Errorf("get/create session: %w", err)
	}
	if isNew {
		a.logger.Info().Str("user_id", userID).Str("session_id", sessionID).Msg("[telegram] created new session after idle timeout")
	}

	normalized := &orchestrator.NormalizedInboundMessage{
		UserID:      userID,
		PlatformID:  fmt.Sprintf("telegram:%s", platformUserID),
		SessionID:   sessionID,
		MessageID:   fmt.Sprintf("%d", msg.ID),
		Text:        msg.Text,
		Attachments: nil,
		Metadata:    make(map[string]any),
		Timestamp:   msg.Time(),
	}

	// Handle Attachments
	if msg.Voice != nil {
		audioData, err := a.downloadFile(msg.Voice.File)
		if err != nil {
			return nil, fmt.Errorf("download voice: %w", err)
		}

		text, err := a.transcription.Transcribe(ctx, audioData)
		if err != nil {
			return nil, fmt.Errorf("transcribe voice: %w", err)
		}

		normalized.Text = text
		normalized.Attachments = append(normalized.Attachments, orchestrator.Attachment{
			Type:        orchestrator.AttachmentTypeVoice,
			Data:        audioData,
			MimeType:    msg.Voice.MIME,
			Transcribed: text,
		})

	} else if msg.Document != nil {
		docData, err := a.downloadFile(msg.Document.File)
		if err != nil {
			return nil, fmt.Errorf("download document: %w", err)
		}

		text, err := a.extraction.ExtractText(ctx, docData, msg.Document.MIME)
		if err != nil {
			return nil, fmt.Errorf("extract document text: %w", err)
		}

		normalized.Attachments = append(normalized.Attachments, orchestrator.Attachment{
			Type:          orchestrator.AttachmentTypeDocument,
			Data:          docData,
			MimeType:      msg.Document.MIME,
			FileName:      msg.Document.FileName,
			ExtractedText: text,
		})

	} else if msg.Photo != nil {
		// Photo in Telegram can have multiple sizes, pick the largest
		photo := msg.Photo
		photoData, err := a.downloadFile(photo.File)
		if err != nil {
			return nil, fmt.Errorf("download photo: %w", err)
		}

		// Try to extract text using vision model (same as WhatsApp)
		text, err := a.extraction.ExtractText(ctx, photoData, "image/jpeg")
		if err != nil {
			a.logger.Warn().Err(err).Msg("[telegram] photo text extraction failed")
			// continue anyway
		} else if text != "" {
			normalized.Text = text
			if msg.Caption != "" {
				normalized.Text = fmt.Sprintf("%s\n\nUser caption: %s", text, msg.Caption)
			}
		}

		normalized.Attachments = append(normalized.Attachments, orchestrator.Attachment{
			Type:          orchestrator.AttachmentTypeImage,
			Data:          photoData,
			MimeType:      "image/jpeg",
			Caption:       msg.Caption,
			ExtractedText: text,
		})
	}

	return normalized, nil
}

func (a *telegramInboundAdapter) downloadFile(file tele.File) ([]byte, error) {
	rc, err := a.bot.File(&file)
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	
	return io.ReadAll(rc)
}

// ─── Outbound Adapter ──────────────────────────────────────────────

type telegramOutboundAdapter struct {
	bot    *tele.Bot
	logger *zerolog.Logger
	buffer strings.Builder
}

func NewOutboundAdapter(bot *tele.Bot, logger *zerolog.Logger) orchestrator.OutboundAdapter {
	return &telegramOutboundAdapter{
		bot:    bot,
		logger: logger,
	}
}

func (a *telegramOutboundAdapter) Deliver(ctx context.Context, sessionID string, chunk orchestrator.ResponseChunk) error {
	switch chunk.Type {
	case orchestrator.ChunkStageEvent:
		// Send chat action (typing...)
		recipientRaw := ctx.Value("tg_recipient")
		if recipient, ok := recipientRaw.(tele.Recipient); ok {
			_ = a.bot.Notify(recipient, tele.Typing)
		}
		return nil

	case orchestrator.ChunkToken:
		// Buffer tokens
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

		recipientRaw := ctx.Value("tg_recipient")
		recipient, ok := recipientRaw.(tele.Recipient)
		if !ok {
			return fmt.Errorf("tg_recipient not found in context")
		}

		// Escape Markdown if using tele.ModeMarkdownV2
		// Or send as plain text
		_, err := a.bot.Send(recipient, finalText, &tele.SendOptions{
			ParseMode: tele.ModeMarkdown,
		})
		
		if err != nil {
			// Fallback to plain text if markdown fails
			_, err = a.bot.Send(recipient, finalText)
		}
		
		return err

	case orchestrator.ChunkError:
		if chunk.Error != nil {
			a.logger.Error().Err(chunk.Error).Msg("[telegram] chunk error received")
		}
		return chunk.Error

	default:
		return nil
	}
}
