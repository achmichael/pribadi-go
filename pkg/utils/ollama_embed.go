package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// OllamaEmbeddingRequest represents the request to Ollama embeddings API
type OllamaEmbeddingRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

// OllamaEmbeddingResponse represents the response from Ollama embeddings API
type OllamaEmbeddingResponse struct {
	Embedding []float32 `json:"embedding"`
}

// OllamaEmbedder provides embedding functionality using Ollama
type OllamaEmbedder struct {
	baseURL string
	model   string
	client  *http.Client
}

// NewOllamaEmbedder creates a new Ollama embedder
func NewOllamaEmbedder(baseURL, model string) *OllamaEmbedder {
	return &OllamaEmbedder{
		baseURL: baseURL,
		model:   model,
		client:  &http.Client{},
	}
}

// Embed generates embeddings for the given text using Ollama
func (e *OllamaEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	reqBody := OllamaEmbeddingRequest{
		Model:  e.model,
		Prompt: text,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", e.baseURL+"/api/embeddings", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama API returned status %d", resp.StatusCode)
	}

	var embeddingResp OllamaEmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&embeddingResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return embeddingResp.Embedding, nil
}
