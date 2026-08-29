package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rs/zerolog"
)

type AnthropicClient struct {
	apiKey     string
	model      string
	logger     *zerolog.Logger
	httpClient *http.Client
}

func NewAnthropicClient(apiKey, model string, logger *zerolog.Logger) *AnthropicClient {
	if model == "" {
		model = "claude-3-haiku-20240307"
	}
	return &AnthropicClient{
		apiKey: apiKey,
		model:  model,
		logger: logger,
		httpClient: &http.Client{
			Timeout: 2 * time.Minute,
		},
	}
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	Messages  []anthropicMessage `json:"messages"`
}

type anthropicResponse struct {
	Type  string `json:"type"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content,omitempty"`
}

func (c *AnthropicClient) Chat(ctx context.Context, messages []ChatMessage) (string, error) {
	var antMessages []anthropicMessage
	for _, m := range messages {
		// Anthropic only supports "user" and "assistant" roles natively in messages.
		// System prompt is handled differently in their API, but for simplicity here we map "system" to "user"
		// or ideally we should separate it. For test connection, user is enough.
		role := m.Role
		if role == "system" {
			role = "user" // fallback
		}
		antMessages = append(antMessages, anthropicMessage{
			Role:    role,
			Content: m.Content,
		})
	}

	reqBody := anthropicRequest{
		Model:     c.model,
		MaxTokens: c.NumPredict(),
		Messages:  antMessages,
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(data))
	if err != nil {
		return "", err
	}

	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("content-type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("network error: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)

	var antResp anthropicResponse
	if err := json.Unmarshal(bodyBytes, &antResp); err != nil {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(bodyBytes))
	}

	if resp.StatusCode != http.StatusOK {
		if antResp.Error != nil {
			return "", fmt.Errorf("HTTP %d (%s): %s", resp.StatusCode, antResp.Error.Type, antResp.Error.Message)
		}
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(bodyBytes))
	}

	if len(antResp.Content) > 0 {
		return antResp.Content[0].Text, nil
	}

	return "", errors.New("empty content from anthropic")
}

func (c *AnthropicClient) SimplyChat(ctx context.Context, prompt string) (string, error) {
	return c.Chat(ctx, []ChatMessage{{Role: "user", Content: prompt}})
}

func (c *AnthropicClient) ChatJSON(ctx context.Context, messages []ChatMessage) (string, error) {
	return c.Chat(ctx, messages)
}

func (c *AnthropicClient) ChatWithTools(ctx context.Context, messages []ChatMessage, format string) (*ChatResult, error) {
	return nil, fmt.Errorf("Anthropic ChatWithTools not implemented in this adapter")
}

func (c *AnthropicClient) ChatWithToolsDirect(ctx context.Context, messages []ChatMessage, format string, tools []Tool) (*ChatResult, error) {
	return nil, fmt.Errorf("Anthropic ChatWithToolsDirect not implemented in this adapter")
}

func (c *AnthropicClient) ChatStream(ctx context.Context, messages []ChatMessage, tools []Tool) (<-chan StreamChunk, error) {
	return nil, fmt.Errorf("Anthropic ChatStream not implemented in this adapter")
}

func (c *AnthropicClient) ChatStreamWithThink(ctx context.Context, messages []ChatMessage, tools []Tool, think bool) (<-chan StreamChunk, error) {
	return nil, fmt.Errorf("Anthropic ChatStreamWithThink not implemented in this adapter")
}

func (c *AnthropicClient) Schemas() []Tool {
	return nil
}

func (c *AnthropicClient) NumCtx() int {
	return 200000
}

func (c *AnthropicClient) NumPredict() int {
	return 4096
}

func (c *AnthropicClient) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	return nil, fmt.Errorf("Anthropic does not support embeddings via this API yet")
}

func (c *AnthropicClient) SimplyChatWithOptions(ctx context.Context, prompt string, opts SimplyChatOptions) (string, error) {
	return c.SimplyChat(ctx, prompt)
}

func (c *AnthropicClient) GenerateChatTitle(ctx context.Context, message string) (string, error) {
	return c.SimplyChat(ctx, message)
}
