package rest

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/achmichael/pribadi-go/internal/domain"
	"github.com/achmichael/pribadi-go/internal/logger"
	"github.com/achmichael/pribadi-go/internal/orchestrator"
	"github.com/google/uuid"
)

func (s *Server) handleChatStream(w http.ResponseWriter, r *http.Request) {
	var log = logger.New("info")

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	var req struct {
		Message   string `json:"message"`
		SessionID string `json:"session_id"`
		Model     string `json:"model"`
		FileJobID string `json:"file_job_id"`
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

	rawEvent := WebInboundRawEvent{
		UserID:    userID,
		SessionID: req.SessionID,
		Message:   req.Message,
		Model:     req.Model,
		FileJobID: req.FileJobID,
	}

	normalized, err := s.webInboundAdapter.Normalize(ctx, rawEvent)
	if err != nil {
		sendSSEEvent(w, flusher, "error", map[string]string{"message": err.Error()})
		log.Error().Err(err).Msg("[rest] normalization failed")
		return
	}

	outAdapter, streamCh := NewOutboundAdapter(s.logger)

	// Pisahkan konteks untuk background processing agar tidak terganggu oleh HTTP client disconnect
	bgCtx := context.WithoutCancel(ctx)

	go func() {
		defer close(streamCh)
		if err := s.coreOrchestrator.HandleMessage(bgCtx, normalized, outAdapter); err != nil {
			log.Error().Err(err).Msg("[rest] orchestrator failed")
			outAdapter.Deliver(bgCtx, normalized.SessionID, orchestrator.ResponseChunk{
				Type:  orchestrator.ChunkError,
				Error: err,
			})
		}
	}()

	for chunk := range streamCh {
		select {
		case <-ctx.Done():
			sendSSEEvent(w, flusher, "interrupted", map[string]bool{"interrupted": true})
			return
		default:
			switch chunk.Type {
			case orchestrator.ChunkError:
				sendSSEEvent(w, flusher, "error", map[string]string{"message": chunk.Error.Error()})
				return

			case orchestrator.ChunkStageEvent:
				sendSSEEvent(w, flusher, "stage", map[string]string{"stage": chunk.Stage})

			case orchestrator.ChunkToolCall:
				sendSSEEvent(w, flusher, "tool", map[string]string{"tool": chunk.ToolName})

			case orchestrator.ChunkToken:
				sendSSEEvent(w, flusher, "token", map[string]string{"content": chunk.Text})

			case orchestrator.ChunkDone:
				if chunk.Text != "" {
					sendSSEEvent(w, flusher, "token", map[string]string{"content": chunk.Text})
				}
				sendSSEEvent(w, flusher, "done", nil)

				s.saveAssistantMessage(context.Background(), normalized.SessionID, chunk.Text, req.Model)
				return
			}
		}
	}
}

func (s *Server) saveAssistantMessage(ctx context.Context, sessionID, content, model string) {
	_ = s.webChatRepo.CreateMessage(ctx, domain.WebChatMessage{
		ID:        uuid.New().String(),
		SessionID: sessionID,
		Role:      "assistant",
		Content:   content,
		Model:     model,
		CreatedAt: time.Now(),
	})
}

func sendSSEEvent(w http.ResponseWriter, flusher http.Flusher, eventType string, data interface{}) {
	bytes, _ := json.Marshal(data)
	w.Write([]byte("event: " + eventType + "\n"))
	w.Write([]byte("data: " + string(bytes) + "\n\n"))
	flusher.Flush()
}
