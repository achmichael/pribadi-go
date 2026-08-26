package usecase

import (
	"context"
	"fmt"

	"github.com/achmichael/pribadi-go/internal/delivery/whatsapp"
	"github.com/achmichael/pribadi-go/internal/orchestrator"
	"github.com/rs/zerolog"
)

type Orchestrator interface {
	Handle(ctx context.Context, msg whatsapp.IncomingMessage) error
}

type orchestratorAdapter struct {
	coreOrchestrator orchestrator.CoreOrchestrator
	inboundAdapter   orchestrator.InboundAdapter
	outboundAdapter  orchestrator.OutboundAdapter
	logger           *zerolog.Logger
}

func NewOrchestrator(
	core orchestrator.CoreOrchestrator,
	inbound orchestrator.InboundAdapter,
	outbound orchestrator.OutboundAdapter,
	logger *zerolog.Logger,
) Orchestrator {
	return &orchestratorAdapter{
		coreOrchestrator: core,
		inboundAdapter:   inbound,
		outboundAdapter:  outbound,
		logger:           logger,
	}
}

func (o *orchestratorAdapter) Handle(ctx context.Context, msg whatsapp.IncomingMessage) error {
	normalized, err := o.inboundAdapter.Normalize(ctx, msg)
	if err != nil {
		return fmt.Errorf("inbound normalize: %w", err)
	}

	ctx = context.WithValue(ctx, "wa_sender_jid", msg.SenderJID)

	return o.coreOrchestrator.HandleMessage(ctx, normalized, o.outboundAdapter)
}

