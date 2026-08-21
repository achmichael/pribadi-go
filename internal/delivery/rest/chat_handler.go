package rest

import (
	"encoding/json"
	"net/http"

	"github.com/achmichael/pribadi-go/internal/logger"
)

// handleChatStream handles SSE streaming for chat completion
func (s *Server) handleChatStream(w http.ResponseWriter, r *http.Request) {
	var log = logger.New("info")
	
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	var req struct {
		Message   string `json:"message"`
		SessionID string `json:"session_id"`
		Model     string `json:"model"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	
	if req.Model == "" {
		req.Model = "local"
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}
	
	userID := r.Context().Value("user_id").(string)
	ctx := r.Context()
	
	stream, err := s.webChatService.StreamChat(ctx, userID, req.SessionID, req.Message, req.Model)
	if err != nil {
		sendSSEEvent(w, flusher, "error", map[string]string{"message": err.Error()})
		log.Info().Int("rest", len(err.Error())).Msg("[error]" + err.Error())
		return
	}

	for chunk := range stream {
		select {
		case <-ctx.Done():
			return // Client disconnected
		default:
			if chunk.Err != nil {
				sendSSEEvent(w, flusher, "error", map[string]string{"message": chunk.Err.Error()})
				return
			}
			
			if len(chunk.ToolCalls) > 0 {
				sendSSEEvent(w, flusher, "tool", chunk.ToolCalls)
			}
			
			if chunk.Content != "" {
				sendSSEEvent(w, flusher, "token", map[string]string{"content": chunk.Content})
			}
			
			if chunk.Done {
				sendSSEEvent(w, flusher, "done", nil)
				return
			}
		}
	}
}

func sendSSEEvent(w http.ResponseWriter, flusher http.Flusher, eventType string, data interface{}) {
	bytes, _ := json.Marshal(data)
	w.Write([]byte("event: " + eventType + "\n"))
	w.Write([]byte("data: " + string(bytes) + "\n\n"))
	flusher.Flush()
}
