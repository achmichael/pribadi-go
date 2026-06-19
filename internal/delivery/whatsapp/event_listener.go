package whatsapp

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types/events"
)

// MessageType represents the type of incoming message
type MessageType string

const (
	MessageTypeText      MessageType = "text"
	MessageTypeImage     MessageType = "image"
	MessageTypeDocument  MessageType = "document"
	MessageTypeVoiceNote MessageType = "voice_note"
	MessageTypeSticker   MessageType = "sticker"
	MessageTypeUnknown   MessageType = "unknown"
)

// IncomingMessage represents an incoming WhatsApp message
type IncomingMessage struct {
	ID          string
	SenderJID   string
	MessageType MessageType
	Timestamp   time.Time

	// Text content (for text messages)
	TextContent string

	// Media info (for media messages)
	MediaURL      string
	MediaMimetype string
	MediaSHA256   []byte
	MediaSize     uint64
	Caption       string

	// Raw message for advanced processing
	RawMessage *waE2E.Message
}

// MessageRouter defines the interface for routing different message types
type MessageRouter interface {
	TextMessage(ctx context.Context, msg IncomingMessage) error
	VoiceMessage(ctx context.Context, msg IncomingMessage) error
	MediaMessage(ctx context.Context, msg IncomingMessage) error
}

// EventListener handles WhatsApp events
type EventListener struct {
	client *whatsmeow.Client
	router MessageRouter
	logger *zerolog.Logger
}

// NewEventListener creates a new event listener
func NewEventListener(client *whatsmeow.Client, router MessageRouter, logger *zerolog.Logger) *EventListener {
	return &EventListener{
		client: client,
		router: router,
		logger: logger,
	}
}

// Start registers event handlers and starts listening
func (l *EventListener) Start() {
	l.client.AddEventHandler(l.handleEvent)
	l.logger.Info().Msg("Event listener started")
}

// handleEvent processes incoming WhatsApp events
func (l *EventListener) handleEvent(evt interface{}) {
	switch v := evt.(type) {
	case *events.Message:
		go l.handleMessage(v)
	}
}

// handleMessage processes incoming messages
func (l *EventListener) handleMessage(evt *events.Message) {
	ctx := context.Background()

	// Extract message info
	msg := l.extractMessage(evt)

	// Log incoming message
	logEvent := l.logger.Info().
		Str("sender", msg.SenderJID).
		Str("type", string(msg.MessageType)).
		Str("id", msg.ID)

	// Add truncated content for logging
	if msg.TextContent != "" {
		truncated := msg.TextContent
		if len(truncated) > 50 {
			truncated = truncated[:50] + "..."
		}
		logEvent.Str("content", truncated)
	} else if msg.Caption != "" {
		truncated := msg.Caption
		if len(truncated) > 50 {
			truncated = truncated[:50] + "..."
		}
		logEvent.Str("caption", truncated)
	}

	logEvent.Msg("Incoming message")

	// Route to appropriate handler
	var err error
	switch msg.MessageType {
	case MessageTypeText:
		err = l.router.TextMessage(ctx, msg)
	case MessageTypeVoiceNote:
		err = l.router.VoiceMessage(ctx, msg)
	case MessageTypeImage, MessageTypeDocument, MessageTypeSticker:
		err = l.router.MediaMessage(ctx, msg)
	default:
		l.logger.Warn().Str("type", string(msg.MessageType)).Msg("Unknown message type")
		return
	}

	if err != nil {
		l.logger.Error().Err(err).Str("type", string(msg.MessageType)).Msg("Failed to route message")
	}
}

// extractMessage extracts relevant information from WhatsApp event
func (l *EventListener) extractMessage(evt *events.Message) IncomingMessage {
	msg := IncomingMessage{
		ID:         evt.Info.ID,
		SenderJID:  evt.Info.Sender.String(),
		Timestamp:  evt.Info.Timestamp,
		RawMessage: evt.Message,
	}

	// Determine message type and extract content
	switch {
	case evt.Message.GetConversation() != "":
		msg.MessageType = MessageTypeText
		msg.TextContent = evt.Message.GetConversation()

	case evt.Message.GetExtendedTextMessage() != nil:
		msg.MessageType = MessageTypeText
		msg.TextContent = evt.Message.GetExtendedTextMessage().GetText()

	case evt.Message.GetImageMessage() != nil:
		msg.MessageType = MessageTypeImage
		imgMsg := evt.Message.GetImageMessage()
		msg.Caption = imgMsg.GetCaption()
		msg.MediaURL = imgMsg.GetURL()
		msg.MediaMimetype = imgMsg.GetMimetype()
		msg.MediaSHA256 = imgMsg.GetFileSHA256()
		msg.MediaSize = imgMsg.GetFileLength()

	case evt.Message.GetDocumentMessage() != nil:
		msg.MessageType = MessageTypeDocument
		docMsg := evt.Message.GetDocumentMessage()
		msg.Caption = docMsg.GetCaption()
		msg.MediaURL = docMsg.GetURL()
		msg.MediaMimetype = docMsg.GetMimetype()
		msg.MediaSHA256 = docMsg.GetFileSHA256()
		msg.MediaSize = docMsg.GetFileLength()

	case evt.Message.GetAudioMessage() != nil:
		audioMsg := evt.Message.GetAudioMessage()
		if audioMsg.GetPTT() {
			msg.MessageType = MessageTypeVoiceNote
		} else {
			msg.MessageType = MessageTypeDocument
		}
		msg.MediaURL = audioMsg.GetURL()
		msg.MediaMimetype = audioMsg.GetMimetype()
		msg.MediaSHA256 = audioMsg.GetFileSHA256()
		msg.MediaSize = audioMsg.GetFileLength()

	case evt.Message.GetStickerMessage() != nil:
		msg.MessageType = MessageTypeSticker
		stickerMsg := evt.Message.GetStickerMessage()
		msg.MediaURL = stickerMsg.GetURL()
		msg.MediaMimetype = stickerMsg.GetMimetype()
		msg.MediaSHA256 = stickerMsg.GetFileSHA256()
		msg.MediaSize = stickerMsg.GetFileLength()

	default:
		msg.MessageType = MessageTypeUnknown
	}

	return msg
}

// DownloadMedia downloads media from a message
func (l *EventListener) DownloadMedia(ctx context.Context, msg IncomingMessage) ([]byte, error) {
	if msg.RawMessage == nil {
		return nil, fmt.Errorf("raw message is nil")
	}

	// Download based on message type
	switch msg.MessageType {
	case MessageTypeImage:
		return l.client.Download(ctx, msg.RawMessage.GetImageMessage())
	case MessageTypeDocument:
		return l.client.Download(ctx, msg.RawMessage.GetDocumentMessage())
	case MessageTypeVoiceNote:
		return l.client.Download(ctx, msg.RawMessage.GetAudioMessage())
	case MessageTypeSticker:
		return l.client.Download(ctx, msg.RawMessage.GetStickerMessage())
	default:
		return nil, fmt.Errorf("unsupported message type for download: %s", msg.MessageType)
	}
}
