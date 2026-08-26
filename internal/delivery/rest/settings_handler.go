package rest

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// handleGetChatSettings returns current settings
func (s *Server) handleGetChatSettings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := ctx.Value("user_id").(string)

	config, _ := s.service.GetConfigByKey(ctx, "system_prompt_template")
	systemPrompt := ""
	if config != nil {
		systemPrompt = config.ValueJSON
	}
	
	// We only return whether the keys exist or not, never the actual keys for security
	openAIKey, _ := s.webChatService.GetAPIKey(ctx, userID, "openai")
	anthropicKey, _ := s.webChatService.GetAPIKey(ctx, userID, "anthropic")

	hasOpenAIKey := openAIKey != ""
	hasAnthropicKey := anthropicKey != ""

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"system_prompt":     systemPrompt,
		"has_openai_key":    hasOpenAIKey,
		"has_anthropic_key": hasAnthropicKey,
	})
}

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

func (s *Server) handleListUploads(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(string)
	jobs, err := s.webChatService.ListUploadJobs(r.Context(), userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to fetch uploads")
		return
	}
	respondJSON(w, http.StatusOK, jobs)
}

func (s *Server) handleDeleteUpload(w http.ResponseWriter, r *http.Request) {
	jobID := chi.URLParam(r, "id")
	userID := r.Context().Value("user_id").(string)
	
	if err := s.webChatService.DeleteUploadJob(r.Context(), userID, jobID); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to delete upload")
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

func (s *Server) handlePurgeUploads(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(string)
	if err := s.webChatService.PurgeUserStorage(r.Context(), userID); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to purge storage")
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "success"})
}
