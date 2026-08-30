package monitor

import (
	"context"
	"fmt"

	"github.com/achmichael/pribadi-go/internal/delivery/whatsapp"
	"github.com/rs/zerolog"
)

type WhatsAppDelivery struct {
	waClient *whatsapp.Client
	logger   *zerolog.Logger
}

func NewWhatsAppDelivery(waClient *whatsapp.Client, logger *zerolog.Logger) *WhatsAppDelivery {
	return &WhatsAppDelivery{waClient: waClient, logger: logger}
}

func (d *WhatsAppDelivery) Send(ctx context.Context, platform string, userID string, message string) error {
	switch platform {
	case "whatsapp":
		return d.waClient.SendText(ctx, userID, message)
	default:
		d.logger.Warn().Str("platform", platform).Msg("unsupported delivery platform, falling back to whatsapp")
		return d.waClient.SendText(ctx, userID, message)
	}
}

type MultiDelivery struct {
	waClient *whatsapp.Client
	logger   *zerolog.Logger
}

func NewMultiDelivery(waClient *whatsapp.Client, logger *zerolog.Logger) *MultiDelivery {
	return &MultiDelivery{waClient: waClient, logger: logger}
}

func (d *MultiDelivery) Send(ctx context.Context, platform string, userID string, message string) error {
	switch platform {
	case "whatsapp":
		return d.waClient.SendText(ctx, userID, message)
	default:
		d.logger.Warn().Str("platform", platform).Str("user", userID).Msg("platform not supported for monitor delivery")
		return fmt.Errorf("unsupported platform: %s", platform)
	}
}
