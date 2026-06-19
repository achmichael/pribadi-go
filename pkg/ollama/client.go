package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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
		baseURL:    baseURL,
		model:      model,
		httpClient: &http.Client{Timeout: 120 * time.Second},
	}
}

// Chat sends chat messages to Ollama
func (c *OllamaClient) Chat(ctx context.Context, messages []ChatMessage) (string, error) {
	// Log token estimate
	totalWords := 0
	for _, m := range messages {
		totalWords += len(strings.Fields(m.Content))
	}
	// Estimation: 1 word ~ 1.3 tokens
	fmt.Printf("Prompt token estimate: %d\n", int(float64(totalWords)*1.3))

	reqBody := chatRequest{
		Model:    c.model,
		Messages: messages,
		Stream:   false,
	}

	data, _ := json.Marshal(reqBody)
	
	// Perform request with simple retry
	var resp *http.Response
	var err error
	for i := 0; i < 2; i++ {
		req, _ := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/chat", bytes.NewBuffer(data))
		req.Header.Set("Content-Type", "application/json")
		
		resp, err = c.httpClient.Do(req)
		if err == nil && resp.StatusCode == 503 && i == 0 {
			resp.Body.Close()
			time.Sleep(1 * time.Second)
			continue
		}
		break
	}
	
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ollama chat error: status %d", resp.StatusCode)
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
