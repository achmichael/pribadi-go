package rest

import (
	"encoding/json"
	"net/http"
)

// handleChatSettings handles saving BYOK and custom system prompts
func (s *Server) handleChatSettings(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SystemPrompt string `json:"system_prompt,omitempty"`
		OpenAIKey    string `json:"openai_key,omitempty"`
		AnthropicKey string `json:"anthropic_key,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	userID := r.Context().Value("user_id").(string)
	ctx := r.Context()

	if req.SystemPrompt != "" {
		// Save to AgentConfig or per-user preference. 
		// For now using AgentConfig as per original design.
		_ = s.service.UpdateConfig(ctx, "system_prompt_template", req.SystemPrompt)
	}

	if req.OpenAIKey != "" {
		_ = s.webChatService.SaveAPIKey(ctx, userID, "openai", req.OpenAIKey)
	}

	if req.AnthropicKey != "" {
		_ = s.webChatService.SaveAPIKey(ctx, userID, "anthropic", req.AnthropicKey)
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "success"})
}
