package orchestrator

import "context"

type InboundAdapter interface {
	Normalize(ctx context.Context, rawEvent any) (*NormalizedInboundMessage, error)
}

type OutboundAdapter interface {
	Deliver(ctx context.Context, sessionID string, chunk ResponseChunk) error
}

type CoreOrchestrator interface {
	HandleMessage(ctx context.Context, msg *NormalizedInboundMessage, out OutboundAdapter) error
}
