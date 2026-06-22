package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
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
		client: &http.Client{
			Timeout: 2 * time.Minute,
		},
	}
}

// Embed generates embeddings for the given text using Ollama
func (e *OllamaEmbedder) Embed(_ context.Context, text string) ([]float32, error) {
	start := time.Now()

	reqBody := OllamaEmbeddingRequest{
		Model:  e.model,
		Prompt: text,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Use background context for the HTTP call since we manage timeout via client
	req, err := http.NewRequest("POST", e.baseURL+"/api/embeddings", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("embed request failed (text_len=%d, elapsed=%s): %w",
			len(text), time.Since(start), err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama embedding API returned status %d (text_len=%d, elapsed=%s)",
			resp.StatusCode, len(text), time.Since(start))
	}

	var embeddingResp OllamaEmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&embeddingResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	_ = time.Since(start) // available for caller logging

	return embeddingResp.Embedding, nil
}
