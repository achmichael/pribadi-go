package whatsapp

import (
	"context"
	"testing"

	"github.com/rs/zerolog"
)

// MockMessageRouter is a mock implementation for testing
type MockMessageRouter struct {
	TextMessages  []IncomingMessage
	VoiceMessages []IncomingMessage
	MediaMessages []IncomingMessage
}

func (m *MockMessageRouter) TextMessage(ctx context.Context, msg IncomingMessage) error {
	m.TextMessages = append(m.TextMessages, msg)
	return nil
}

func (m *MockMessageRouter) VoiceMessage(ctx context.Context, msg IncomingMessage) error {
	m.VoiceMessages = append(m.VoiceMessages, msg)
	return nil
}

func (m *MockMessageRouter) MediaMessage(ctx context.Context, msg IncomingMessage) error {
	m.MediaMessages = append(m.MediaMessages, msg)
	return nil
}

func TestMessageTypes(t *testing.T) {
	tests := []struct {
		name     string
		msgType  MessageType
		expected string
	}{
		{"text", MessageTypeText, "text"},
		{"image", MessageTypeImage, "image"},
		{"document", MessageTypeDocument, "document"},
		{"voice_note", MessageTypeVoiceNote, "voice_note"},
		{"sticker", MessageTypeSticker, "sticker"},
		{"unknown", MessageTypeUnknown, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.msgType) != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, tt.msgType)
			}
		})
	}
}

func TestMockMessageRouter(t *testing.T) {
	router := &MockMessageRouter{}
	logger := zerolog.Nop()
	ctx := context.Background()

	// Test text message routing
	textMsg := IncomingMessage{
		ID:          "test-1",
		SenderJID:   "1234567890@s.whatsapp.net",
		MessageType: MessageTypeText,
		TextContent: "Hello, World!",
	}

	err := router.TextMessage(ctx, textMsg)
	if err != nil {
		t.Fatalf("Failed to route text message: %v", err)
	}

	if len(router.TextMessages) != 1 {
		t.Errorf("Expected 1 text message, got %d", len(router.TextMessages))
	}

	// Test voice message routing
	voiceMsg := IncomingMessage{
		ID:          "test-2",
		SenderJID:   "1234567890@s.whatsapp.net",
		MessageType: MessageTypeVoiceNote,
		MediaURL:    "https://example.com/voice.ogg",
	}

	err = router.VoiceMessage(ctx, voiceMsg)
	if err != nil {
		t.Fatalf("Failed to route voice message: %v", err)
	}

	if len(router.VoiceMessages) != 1 {
		t.Errorf("Expected 1 voice message, got %d", len(router.VoiceMessages))
	}

	// Test media message routing
	mediaMsg := IncomingMessage{
		ID:          "test-3",
		SenderJID:   "1234567890@s.whatsapp.net",
		MessageType: MessageTypeImage,
		MediaURL:    "https://example.com/image.jpg",
		Caption:     "Test image",
	}

	err = router.MediaMessage(ctx, mediaMsg)
	if err != nil {
		t.Fatalf("Failed to route media message: %v", err)
	}

	if len(router.MediaMessages) != 1 {
		t.Errorf("Expected 1 media message, got %d", len(router.MediaMessages))
	}

	_ = logger // Suppress unused variable warning
}
