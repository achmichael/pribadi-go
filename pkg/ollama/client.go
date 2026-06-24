package ollama

import (
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

// ChatMessage represents a message in the conversation
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model     string            `json:"model"`
	Messages  []ChatMessage     `json:"messages"`
	Stream    bool              `json:"stream"`
	Format    string            `json:"format,omitempty"`
	Options   map[string]any    `json:"options,omitempty"`
	KeepAlive string            `json:"keep_alive,omitempty"`
}

type chatResponse struct {
	Message ChatMessage `json:"message"`
}

type embeddingRequest struct {
	Model     string `json:"model"`
	Prompt    string `json:"prompt"`
	KeepAlive string `json:"keep_alive,omitempty"`
}

type embeddingResponse struct {
	Embedding []float32 `json:"embedding"`
}

// OllamaClient handles communication with Ollama API
type OllamaClient struct {
	baseURL    string
	model      string
	httpClient *http.Client
	logger     *zerolog.Logger
}

// NewClient creates a new Ollama client
func NewClient(baseURL, model string, logger *zerolog.Logger) *OllamaClient {
	return &OllamaClient{
		baseURL: baseURL,
		model:   model,
		httpClient: &http.Client{Timeout: 5 * time.Minute},
		logger:  logger,
	}
}

// MaxPromptTokens is the soft cap for total prompt tokens sent to Chat.
const MaxPromptTokens = 2048

// estimateTokens rough count: words * 1.3
func estimateTokens(messages []ChatMessage) int {
	total := 0
	for _, m := range messages {
		total += len(strings.Fields(m.Content))
	}
	return int(float64(total) * 1.3)
}

// truncateToWordLimit trims content to approximately maxWords words.
func truncateToWordLimit(s string, maxWords int) string {
	words := strings.Fields(s)
	if len(words) <= maxWords {
		return s
	}
	return strings.Join(words[:maxWords], " ") + " [truncated]"
}

// Warmup preloads the chat model into Ollama memory.
func (c *OllamaClient) Warmup(ctx context.Context) error {
	_, err := c.Chat(ctx, []ChatMessage{
		{Role: "user", Content: "hi"},
	})
	return err
}

// Chat sends chat messages to Ollama.
func (c *OllamaClient) Chat(ctx context.Context, messages []ChatMessage) (string, error) {
	return c.doChat(ctx, messages, "")
}

// ChatJSON sends chat messages to Ollama and guarantees JSON output.
func (c *OllamaClient) ChatJSON(ctx context.Context, messages []ChatMessage) (string, error) {
	return c.doChat(ctx, messages, "json")
}

func (c *OllamaClient) doChat(ctx context.Context, messages []ChatMessage, format string) (string, error) {
	tokenEst := estimateTokens(messages)
	c.logger.Info().
		Int("token_estimate", tokenEst).
		Str("model", c.model).
		Msg("[ollama] chat request preparing")

	// Guard: if prompt too large, truncate the longest message
	if tokenEst > MaxPromptTokens {
		maxIdx := 0
		maxLen := 0
		for i, m := range messages {
			wc := len(strings.Fields(m.Content))
			if wc > maxLen {
				maxLen = wc
				maxIdx = i
			}
		}
		excess := tokenEst - MaxPromptTokens
		excessWords := int(float64(excess)/1.3) + 10
		targetWords := maxLen - excessWords
		if targetWords < 50 {
			targetWords = 50
		}
		messages[maxIdx].Content = truncateToWordLimit(messages[maxIdx].Content, targetWords)
		newEst := estimateTokens(messages)
		c.logger.Warn().
			Int("original_tokens", tokenEst).
			Int("truncated_tokens", newEst).
			Int("truncated_role_idx", maxIdx).
			Msg("[ollama] prompt truncated to fit MaxPromptTokens")
	}

	reqBody := chatRequest{
		Model:    c.model,
		Messages: messages,
		Stream:   false,
		Format:   format,
		Options: map[string]any{
			"num_predict": 800,  // increased to allow full JSON abstract extraction
			"temperature": 0.7,
			"num_ctx":     2048, // limit context window for speed
		},
		KeepAlive: "30m", // keep model loaded 30 min
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
			return "", fmt.Errorf("ollama chat failed after 3 retries: %w", err)
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
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		c.logger.Error().
			Int("status", resp.StatusCode).
			Str("body", string(body)).
			Msg("[ollama] chat non-OK response")
		return "", fmt.Errorf("ollama chat error: status %d body: %s", resp.StatusCode, string(body))
	}

	decodeStart := time.Now()
	var chatResp chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", fmt.Errorf("decode chat response: %w", err)
	}
	c.logger.Debug().
		Dur("decode_ms", time.Since(decodeStart)).
		Int("reply_len", len(chatResp.Message.Content)).
		Msg("[ollama] chat response decoded")

	return chatResp.Message.Content, nil
}

// GenerateEmbedding generates embeddings for given text.
func (c *OllamaClient) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	start := time.Now()
	textLen := len(text)

	reqBody := embeddingRequest{Model: c.model, Prompt: text, KeepAlive: "30m"}
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
