package llm

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"
)

type GrokClient struct {
	openAIAdapter *OpenAIClient
}

func NewGrokClient(apiKey, model string, logger *zerolog.Logger) *GrokClient {
	if model == "" {
		model = "grok-beta"
	}
	// xAI Grok API is OpenAI compatible, just different base URL
	client := NewOpenAIClientWithBaseURL(apiKey, model, "https://api.x.ai/v1", logger)
	return &GrokClient{
		openAIAdapter: client,
	}
}

// Delegate all to OpenAIAdapter
func (c *GrokClient) Chat(ctx context.Context, messages []ChatMessage) (string, error) {
	return c.openAIAdapter.Chat(ctx, messages)
}

func (c *GrokClient) ChatJSON(ctx context.Context, messages []ChatMessage) (string, error) {
	return c.openAIAdapter.ChatJSON(ctx, messages)
}

func (c *GrokClient) ChatWithTools(ctx context.Context, messages []ChatMessage, format string) (*ChatResult, error) {
	return c.openAIAdapter.ChatWithTools(ctx, messages, format)
}

func (c *GrokClient) ChatWithToolsDirect(ctx context.Context, messages []ChatMessage, format string, tools []Tool) (*ChatResult, error) {
	return c.openAIAdapter.ChatWithToolsDirect(ctx, messages, format, tools)
}

func (c *GrokClient) ChatStream(ctx context.Context, messages []ChatMessage, tools []Tool) (<-chan StreamChunk, error) {
	return c.openAIAdapter.ChatStream(ctx, messages, tools)
}

func (c *GrokClient) ChatStreamWithThink(ctx context.Context, messages []ChatMessage, tools []Tool, think bool) (<-chan StreamChunk, error) {
	return c.openAIAdapter.ChatStreamWithThink(ctx, messages, tools, think)
}

func (c *GrokClient) Schemas() []Tool { return nil }
func (c *GrokClient) NumCtx() int { return 131072 }
func (c *GrokClient) NumPredict() int { return 4096 }
func (c *GrokClient) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	return nil, fmt.Errorf("Grok Embedding not implemented")
}
func (c *GrokClient) SimplyChat(ctx context.Context, prompt string) (string, error) {
	return c.openAIAdapter.SimplyChat(ctx, prompt)
}
func (c *GrokClient) SimplyChatWithOptions(ctx context.Context, prompt string, opts SimplyChatOptions) (string, error) {
	return c.openAIAdapter.SimplyChatWithOptions(ctx, prompt, opts)
}
func (c *GrokClient) GenerateChatTitle(ctx context.Context, message string) (string, error) {
	return c.openAIAdapter.GenerateChatTitle(ctx, message)
}
