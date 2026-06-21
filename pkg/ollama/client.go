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
)

// ChatMessage represents a message in the conversation
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
	Stream   bool          `json:"stream"`
}

type chatResponse struct {
	Message ChatMessage `json:"message"`
}

type embeddingRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

type embeddingResponse struct {
	Embedding []float32 `json:"embedding"`
}

// OllamaClient handles communication with Ollama API
type OllamaClient struct {
	baseURL    string
	model      string
	httpClient *http.Client
}

// NewClient creates a new Ollama client
func NewClient(baseURL, model string) *OllamaClient {
	return &OllamaClient{
		baseURL: baseURL,
		model:   model,
		httpClient: &http.Client{Timeout: 5 * time.Minute},
	}
}

// MaxPromptTokens is the soft cap for total prompt tokens sent to Chat.
// Prompts exceeding this are truncated to avoid timeouts on modest hardware.
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

// Chat sends chat messages to Ollama.
// It enforces MaxPromptTokens by truncating the user message if needed.
func (c *OllamaClient) Chat(ctx context.Context, messages []ChatMessage) (string, error) {
	tokenEst := estimateTokens(messages)
	fmt.Printf("Prompt token estimate: %d\n", tokenEst)

	// Guard: if prompt too large, truncate the longest message (usually user)
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
		// How many words to cut: keep total under MaxPromptTokens
		excess := tokenEst - MaxPromptTokens
		excessWords := int(float64(excess) / 1.3) + 10 // extra margin
		targetWords := maxLen - excessWords
		if targetWords < 50 {
			targetWords = 50
		}
		messages[maxIdx].Content = truncateToWordLimit(messages[maxIdx].Content, targetWords)
		fmt.Printf("Prompt truncated to ~%d tokens (was %d)\n", estimateTokens(messages), tokenEst)
	}

	reqBody := chatRequest{
		Model:    c.model,
		Messages: messages,
		Stream:   false,
	}

	data, _ := json.Marshal(reqBody)

	// Perform request with simple retry
	var resp *http.Response
	var err error
	for i := 0; i < 3; i++ {
		req, _ := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/chat", bytes.NewBuffer(data))
		req.Header.Set("Content-Type", "application/json")

		resp, err = c.httpClient.Do(req)
		if err != nil {
			// Retry on timeout/connection errors
			if i < 2 {
				time.Sleep(time.Duration(i+1) * 2 * time.Second)
				continue
			}
			return "", fmt.Errorf("ollama chat failed after retries: %w", err)
		}
		if resp.StatusCode == 503 && i < 2 {
			resp.Body.Close()
			time.Sleep(time.Duration(i+1) * 2 * time.Second)
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
		return "", fmt.Errorf("ollama chat error: status %d body: %s", resp.StatusCode, string(body))
	}

	var chatResp chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", err
	}

	return chatResp.Message.Content, nil
}

// GenerateEmbedding generates embeddings
func (c *OllamaClient) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	reqBody := embeddingRequest{Model: c.model, Prompt: text}
	data, _ := json.Marshal(reqBody)

	req, _ := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/embeddings", bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var embResp embeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&embResp); err != nil {
		return nil, err
	}
	return embResp.Embedding, nil
}
