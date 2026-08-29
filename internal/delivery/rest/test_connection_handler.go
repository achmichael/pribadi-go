package rest

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/achmichael/pribadi-go/pkg/llm"
)

// handleTestConnection tests a specific LLM connection
func (s *Server) handleTestConnection(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Provider string `json:"provider"`
		ApiKey   string `json:"api_key"`
		Model    string `json:"model"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Provider == "" || req.ApiKey == "" {
		respondError(w, http.StatusBadRequest, "provider and api_key are required")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	var client llm.Client

	if req.Provider == "openai" {
		model := req.Model
		if model == "" {
			model = "gpt-4o-mini"
		}
		client = llm.NewOpenAIClient(req.ApiKey, model, s.logger)
	} else if req.Provider == "anthropic" {
		model := req.Model
		if model == "" {
			model = "claude-3-haiku-20240307"
		}
		client = llm.NewAnthropicClient(req.ApiKey, model, s.logger)
	} else if req.Provider == "gemini" {
		model := req.Model
		if model == "" {
			model = "gemini-1.5-flash"
		}
		client = llm.NewGeminiClient(req.ApiKey, model, s.logger)
	} else if req.Provider == "grok" {
		model := req.Model
		if model == "" {
			model = "grok-beta"
		}
		client = llm.NewGrokClient(req.ApiKey, model, s.logger)
	} else {
		respondError(w, http.StatusBadRequest, "unsupported provider for test")
		return
	}

	// Make a tiny call to verify
	resp, err := client.SimplyChat(ctx, "Hello, reply with 'OK' if you can read this.")
	if err != nil {
		msg := err.Error()
		
		// Map common OpenAI errors to human readable
		if strings.Contains(msg, "401") || strings.Contains(msg, "invalid_api_key") {
			msg = "API Key is invalid or expired."
		} else if strings.Contains(msg, "404") || strings.Contains(msg, "model_not_found") {
			msg = "Model not found or you don't have access to it."
		} else if strings.Contains(msg, "429") || strings.Contains(msg, "insufficient_quota") {
			msg = "Quota exceeded or rate limited. Please check your billing."
		}
		
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"error":   msg,
		})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Connection successful! Model replied: " + resp,
	})
}
