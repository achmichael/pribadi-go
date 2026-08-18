package ollama

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

type ToolRegistry interface {
	Schemas() []Tool
}

type ChatMessage struct {
	Role      string     `json:"role"`
	Content   string     `json:"content"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

type Tool struct {
	Type     string   `json:"type"`
	Function Function `json:"function"`
}

type Function struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

type ToolCall struct {
	Function ToolCallFunction `json:"function"`
}

type ToolCallFunction struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

type chatRequest struct {
	Model     string         `json:"model"`
	Messages  []ChatMessage  `json:"messages"`
	Stream    bool           `json:"stream"`
	Format    string         `json:"format,omitempty"`
	Tools     []Tool         `json:"tools,omitempty"`
	Options   map[string]any `json:"options,omitempty"`
	KeepAlive string         `json:"keep_alive,omitempty"`
}

type chatResponse struct {
	Message ChatMessage `json:"message"`
	Done    bool        `json:"done"`
}

type ChatResult struct {
	Content   string
	ToolCalls []ToolCall
}

type StreamChunk struct {
	Content   string
	ToolCalls []ToolCall
	Done      bool
	Err       error
}

type embeddingRequest struct {
	Model     string `json:"model"`
	Prompt    string `json:"prompt"`
	KeepAlive string `json:"keep_alive,omitempty"`
}

type embeddingResponse struct {
	Embedding []float32 `json:"embedding"`
}

type OllamaClient struct {
	baseURL      string
	model        string
	httpClient   *http.Client
	logger       *zerolog.Logger
	toolRegistry ToolRegistry
}

func NewClient(baseURL, model string, logger *zerolog.Logger, registry ToolRegistry) *OllamaClient {
	return &OllamaClient{
		baseURL:      baseURL,
		model:        model,
		httpClient:   &http.Client{Timeout: 5 * time.Minute},
		logger:       logger,
		toolRegistry: registry,
	}
}

const MaxPromptTokens = 8192

func estimateTokens(messages []ChatMessage) int {
	total := 0
	for _, m := range messages {
		total += len(strings.Fields(m.Content))
	}
	return int(float64(total) * 1.3)
}

func (c *OllamaClient) Warmup(ctx context.Context) error {
	_, err := c.Chat(ctx, []ChatMessage{
		{Role: "user", Content: "hi"},
	})
	return err
}

func (c *OllamaClient) Chat(ctx context.Context, messages []ChatMessage) (string, error) {
	result, err := c.ChatWithTools(ctx, messages, "")
	if err != nil {
		return "", err
	}
	return result.Content, nil
}

func (c *OllamaClient) ChatJSON(ctx context.Context, messages []ChatMessage) (string, error) {
	result, err := c.ChatWithTools(ctx, messages, "json")
	if err != nil {
		return "", err
	}
	return result.Content, nil
}

func (c *OllamaClient) ChatWithTools(ctx context.Context, messages []ChatMessage, format string) (*ChatResult, error) {
	return c.doChatFull(ctx, messages, format, c.toolRegistry.Schemas())
}

func (c *OllamaClient) ChatWithToolsDirect(ctx context.Context, messages []ChatMessage, format string, tools []Tool) (*ChatResult, error) {
	return c.doChatFull(ctx, messages, format, tools)
}

func (c *OllamaClient) doChatFull(ctx context.Context, messages []ChatMessage, format string, tools []Tool) (*ChatResult, error) {
	tokenEst := estimateTokens(messages)
	c.logger.Info().
		Int("token_estimate", tokenEst).
		Str("model", c.model).
		Msg("[ollama] chat request preparing")

	reqBody := chatRequest{
		Model:    c.model,
		Messages: messages,
		Stream:   false,
		Format:   format,
		Tools:    tools,
		Options: map[string]any{
			"num_predict": 800,
			"temperature": 0.7,
			"num_ctx":     4096,
		},
		KeepAlive: "5m",
	}

	data, _ := json.Marshal(reqBody)

	var resp *http.Response
	var err error
	for i := 0; i < 3; i++ {
		attemptStart := time.Now()
		req, _ := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/chat", bytes.NewBuffer(data))
		req.Header.Set("Content-Type", "application/json")

		c.logger.Debug().
			Int("attempt", i+1).
			Int("payload_bytes", len(data)).
			Msg("[ollama] chat HTTP request sending")

		resp, err = c.httpClient.Do(req)

		if err != nil {
			c.logger.Warn().Err(err).
				Int("attempt", i+1).
				Dur("duration_ms", time.Since(attemptStart)).
				Msg("[ollama] chat HTTP request failed")
			if i < 2 {
				backoff := time.Duration(i+1) * 2 * time.Second
				c.logger.Info().Dur("backoff", backoff).Msg("[ollama] retrying after backoff")
				time.Sleep(backoff)
				continue
			}
			return nil, fmt.Errorf("ollama chat failed after 3 retries: %w", err)
		}

		c.logger.Debug().
			Int("attempt", i+1).
			Int("status", resp.StatusCode).
			Dur("duration_ms", time.Since(attemptStart)).
			Msg("[ollama] chat HTTP response received")

		if resp.StatusCode == 503 && i < 2 {
			resp.Body.Close()
			backoff := time.Duration(i+1) * 2 * time.Second
			c.logger.Warn().Dur("backoff", backoff).Msg("[ollama] 503 received, retrying")
			time.Sleep(backoff)
			continue
		}
		break
	}

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		c.logger.Error().
			Int("status", resp.StatusCode).
			Str("body", string(body)).
			Msg("[ollama] chat non-OK response")
		return nil, fmt.Errorf("ollama chat error: status %d body: %s", resp.StatusCode, string(body))
	}

	decodeStart := time.Now()
	var chatResp chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return nil, fmt.Errorf("decode chat response: %w", err)
	}
	c.logger.Debug().
		Dur("decode_ms", time.Since(decodeStart)).
		Int("reply_len", len(chatResp.Message.Content)).
		Int("tool_calls", len(chatResp.Message.ToolCalls)).
		Msg("[ollama] chat response decoded")

	return &ChatResult{
		Content:   chatResp.Message.Content,
		ToolCalls: chatResp.Message.ToolCalls,
	}, nil
}

func (c *OllamaClient) ChatStream(ctx context.Context, messages []ChatMessage, tools []Tool) (<-chan StreamChunk, error) {
	tokenEst := estimateTokens(messages)
	c.logger.Info().
		Int("token_estimate", tokenEst).
		Str("model", c.model).
		Msg("[ollama] stream chat request preparing")

	reqBody := chatRequest{
		Model:    c.model,
		Messages: messages,
		Stream:   true,
		Tools:    tools,
		Options: map[string]any{
			"num_predict": 800,
			"temperature": 0.7,
			"num_ctx":     4096,
		},
		KeepAlive: "5m",
	}

	data, _ := json.Marshal(reqBody)

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/chat", bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("create stream request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("stream request failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("ollama stream error: status %d body: %s", resp.StatusCode, string(body))
	}

	ch := make(chan StreamChunk, 64)
	go func() {
		defer close(ch)
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

		for scanner.Scan() {
			line := scanner.Bytes()
			if len(line) == 0 {
				continue
			}

			var chunk chatResponse
			if err := json.Unmarshal(line, &chunk); err != nil {
				ch <- StreamChunk{Err: fmt.Errorf("decode stream chunk: %w", err)}
				return
			}

			ch <- StreamChunk{
				Content:   chunk.Message.Content,
				ToolCalls: chunk.Message.ToolCalls,
				Done:      chunk.Done,
			}

			if chunk.Done {
				return
			}
		}

		if err := scanner.Err(); err != nil {
			ch <- StreamChunk{Err: fmt.Errorf("stream read: %w", err)}
		}
	}()

	return ch, nil
}

func (c *OllamaClient) Schemas() []Tool {
	if c.toolRegistry != nil {
		return c.toolRegistry.Schemas()
	}
	return nil
}

func (c *OllamaClient) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	start := time.Now()
	textLen := len(text)

	reqBody := embeddingRequest{Model: c.model, Prompt: text, KeepAlive: "10s"}
	data, _ := json.Marshal(reqBody)

	req, _ := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/embeddings", bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error().Err(err).
			Int("text_len", textLen).
			Dur("duration_ms", time.Since(start)).
			Msg("[ollama] embedding request failed")
		return nil, fmt.Errorf("embedding request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		c.logger.Error().
			Int("status", resp.StatusCode).
			Str("body", string(body)).
			Msg("[ollama] embedding non-OK response")
		return nil, fmt.Errorf("ollama embedding error: status %d", resp.StatusCode)
	}

	var embResp embeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&embResp); err != nil {
		return nil, fmt.Errorf("decode embedding response: %w", err)
	}

	c.logger.Debug().
		Int("text_len", textLen).
		Int("embedding_dim", len(embResp.Embedding)).
		Dur("duration_ms", time.Since(start)).
		Msg("[ollama] embedding done")

	return embResp.Embedding, nil
}
