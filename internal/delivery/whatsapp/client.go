package whatsapp

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	"github.com/mdp/qrterminal/v3"
	"github.com/rs/zerolog"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
)

// Client wraps whatsmeow client with simplified methods
type Client struct {
	client *whatsmeow.Client
	logger *zerolog.Logger
}

// waLogger adapts zerolog to whatsmeow logger interface
type waLogger struct {
	logger *zerolog.Logger
}

func (l *waLogger) Errorf(msg string, args ...interface{}) {
	l.logger.Error().Msgf(msg, args...)
}

func (l *waLogger) Warnf(msg string, args ...interface{}) {
	l.logger.Warn().Msgf(msg, args...)
}

func (l *waLogger) Infof(msg string, args ...interface{}) {
	l.logger.Info().Msgf(msg, args...)
}

func (l *waLogger) Debugf(msg string, args ...interface{}) {
	l.logger.Debug().Msgf(msg, args...)
}

func (l *waLogger) Sub(module string) waLog.Logger {
	subLogger := l.logger.With().Str("module", module).Logger()
	return &waLogger{logger: &subLogger}
}

// NewClient creates a new WhatsApp client
func NewClient(db *sql.DB, logger *zerolog.Logger) (*Client, error) {
	// Create whatsmeow logger adapter
	waLogger := &waLogger{logger: logger}

	// Create sqlstore container
	container := sqlstore.NewWithDB(db, "sqlite3", waLogger)

	ctx := context.Background()

	err := container.Upgrade(ctx)
	if err != nil {
		return nil, fmt.Errorf("gagal membuat tabel whatsmeow: %w", err)
	}

	// Get the first device from store or create new one
	deviceStore, err := container.GetFirstDevice(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get device: %w", err)
	}

	// Create whatsmeow client
	client := whatsmeow.NewClient(deviceStore, waLogger)

	return &Client{
		client: client,
		logger: logger,
	}, nil
}

// Connect connects to WhatsApp
func (c *Client) Connect(ctx context.Context) error {
	// Check if already logged in
	if c.client.Store.ID == nil {
		// Not logged in, need to show QR code
		qrChan, _ := c.client.GetQRChannel(ctx)

		err := c.client.Connect()
		if err != nil {
			return fmt.Errorf("failed to connect: %w", err)
		}

		// Wait for QR code and display it
		for evt := range qrChan {
			if evt.Event == "code" {
				c.logger.Info().Msg("QR code received, scan with WhatsApp:")
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
			} else {
				c.logger.Info().Str("event", evt.Event).Msg("QR channel event")
			}
		}
	} else {
		// Already logged in, just connect
		c.logger.Info().Msg("Already logged in, connecting...")
		err := c.client.Connect()
		if err != nil {
			return fmt.Errorf("failed to connect: %w", err)
		}
	}

	c.logger.Info().Msg("Connected to WhatsApp successfully")
	return nil
}

// Disconnect disconnects from WhatsApp
func (c *Client) Disconnect() {
	if c.client != nil {
		c.client.Disconnect()
		c.logger.Info().Msg("Disconnected from WhatsApp")
	}
}

// SendText sends a text message
func (c *Client) SendText(ctx context.Context, jid string, text string) error {
	// Parse JID
	recipient, err := types.ParseJID(jid)
	if err != nil {
		return fmt.Errorf("failed to parse JID: %w", err)
	}

	// Send message
	_, err = c.client.SendMessage(ctx, recipient, &waE2E.Message{
		Conversation: proto.String(text),
	})
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	c.logger.Info().Str("jid", jid).Msg("Text message sent")
	return nil
}

// SendImage sends an image with optional caption
func (c *Client) SendImage(ctx context.Context, jid string, imageBytes []byte, caption string) error {
	// Parse JID
	recipient, err := types.ParseJID(jid)
	if err != nil {
		return fmt.Errorf("failed to parse JID: %w", err)
	}

	// Upload image
	uploaded, err := c.client.Upload(ctx, imageBytes, whatsmeow.MediaImage)
	if err != nil {
		return fmt.Errorf("failed to upload image: %w", err)
	}

	// Prepare image message
	imageMsg := &waE2E.ImageMessage{
		URL:           proto.String(uploaded.URL),
		DirectPath:    proto.String(uploaded.DirectPath),
		MediaKey:      uploaded.MediaKey,
		FileEncSHA256: uploaded.FileEncSHA256,
		FileSHA256:    uploaded.FileSHA256,
		FileLength:    proto.Uint64(uploaded.FileLength),
		Mimetype:      proto.String("image/jpeg"),
	}

	if caption != "" {
		imageMsg.Caption = proto.String(caption)
	}

	// Send message
	_, err = c.client.SendMessage(ctx, recipient, &waE2E.Message{
		ImageMessage: imageMsg,
	})
	if err != nil {
		return fmt.Errorf("failed to send image: %w", err)
	}

	c.logger.Info().Str("jid", jid).Msg("Image message sent")
	return nil
}

// GetClient returns the underlying whatsmeow client for event handling
func (c *Client) GetClient() *whatsmeow.Client {
	return c.client
}
