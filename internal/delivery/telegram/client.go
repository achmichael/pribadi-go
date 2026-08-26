package telegram

import (
	"context"
	"github.com/achmichael/pribadi-go/internal/orchestrator"
	"github.com/rs/zerolog"
	tele "gopkg.in/telebot.v3"
)

type Client struct {
	bot              *tele.Bot
	coreOrchestrator orchestrator.CoreOrchestrator
	inboundAdapter   orchestrator.InboundAdapter
	outboundAdapter  orchestrator.OutboundAdapter
	logger           *zerolog.Logger
}

func NewClient(
	bot *tele.Bot,
	core orchestrator.CoreOrchestrator,
	inbound orchestrator.InboundAdapter,
	outbound orchestrator.OutboundAdapter,
	logger *zerolog.Logger,
) *Client {
	c := &Client{
		bot:              bot,
		coreOrchestrator: core,
		inboundAdapter:   inbound,
		outboundAdapter:  outbound,
		logger:           logger,
	}

	c.setupRoutes()

	return c
}

func (c *Client) setupRoutes() {
	c.bot.Handle(tele.OnText, c.handleMessage)
	c.bot.Handle(tele.OnPhoto, c.handleMessage)
	c.bot.Handle(tele.OnDocument, c.handleMessage)
	c.bot.Handle(tele.OnVoice, c.handleMessage)
}

func (c *Client) handleMessage(tc tele.Context) error {
	ctx := context.Background()
	
	// Inject recipient into context for OutboundAdapter
	ctx = context.WithValue(ctx, "tg_recipient", tc.Sender())

	normalized, err := c.inboundAdapter.Normalize(ctx, tc)
	if err != nil {
		c.logger.Error().Err(err).Msg("[telegram] inbound normalize failed")
		return err
	}

	return c.coreOrchestrator.HandleMessage(ctx, normalized, c.outboundAdapter)
}

func (c *Client) Start() {
	c.logger.Info().Msg("[telegram] Starting bot polling")
	go c.bot.Start()
}

func (c *Client) Stop() {
	c.logger.Info().Msg("[telegram] Stopping bot")
	c.bot.Stop()
}

func (c *Client) GetBot() *tele.Bot {
	return c.bot
}
