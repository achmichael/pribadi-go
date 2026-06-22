package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// ollamaEmbedRequest uses the newer /api/embed endpoint (batch-capable).
type ollamaEmbedRequest struct {
	Model     string `json:"model"`
	Input     string `json:"input"`
	KeepAlive string `json:"keep_alive"`
}

// ollamaEmbedResponse matches /api/embed response.
type ollamaEmbedResponse struct {
	Embeddings [][]float32 `json:"embeddings"`
}

// OllamaEmbedder provides embedding functionality using Ollama.
type OllamaEmbedder struct {
	baseURL   string
	model     string
	client    *http.Client
	cache     sync.Map          // simple query→embedding cache
	keepAlive string            // Ollama keep_alive duration string
}

// NewOllamaEmbedder creates a new Ollama embedder.
func NewOllamaEmbedder(baseURL, model string) *OllamaEmbedder {
	return &OllamaEmbedder{
		baseURL:   baseURL,
		model:     model,
		keepAlive: "30m", // keep model loaded 30 min to avoid cold-start
		client: &http.Client{
			Timeout: 2 * time.Minute,
		},
	}
}

// Warmup preloads the embedding model into Ollama memory so first real
// request doesn't pay cold-start penalty (~60s → <1s).
func (e *OllamaEmbedder) Warmup(ctx context.Context) error {
	_, err := e.Embed(ctx, "warmup")
	return err
}

// Embed generates embeddings for the given text using Ollama /api/embed.
func (e *OllamaEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	// Check cache first (useful for repeated identical queries).
	if cached, ok := e.cache.Load(text); ok {
		return cached.([]float32), nil
	}

	start := time.Now()

	reqBody := ollamaEmbedRequest{
		Model:     e.model,
		Input:     text,
		KeepAlive: e.keepAlive,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", e.baseURL+"/api/embed", bytes.NewBuffer(jsonData))
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
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama embed API status %d (text_len=%d, elapsed=%s): %s",
			resp.StatusCode, len(text), time.Since(start), string(body))
	}

	var embedResp ollamaEmbedResponse
	if err := json.NewDecoder(resp.Body).Decode(&embedResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(embedResp.Embeddings) == 0 {
		return nil, fmt.Errorf("empty embeddings returned (text_len=%d)", len(text))
	}

	result := embedResp.Embeddings[0]

	// Cache result for short queries (likely search queries, not documents).
	if len(text) < 500 {
		e.cache.Store(text, result)
	}

	return result, nil
}
