package rest

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (s *Server) handleGetSessions(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(string)

	sessions, err := s.webChatService.ListSessions(r.Context(), userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, sessions)
}

func (s *Server) handleCreateSession(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(string)

	var req struct {
		Message string `json:"message"`
	}

	_ = json.NewDecoder(r.Body).Decode(&req)

	generatedTitle, err := s.webChatService.GenerateChatTitle(r.Context(), req.Message)

	fmt.Println("generated Title" + generatedTitle)
	
	if err != nil {
		generatedTitle = "New Chat"
	}

	session, err := s.webChatService.CreateSession(r.Context(), userID, generatedTitle)

	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, session)
}

func (s *Server) handleGetSessionHistory(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")
	userID := r.Context().Value("user_id").(string)

	// Verify ownership
	session, err := s.webChatService.GetSession(r.Context(), sessionID)
	if err != nil || session == nil || session.UserID != userID {
		respondError(w, http.StatusNotFound, "session not found")
		return
	}

	history, err := s.webChatService.GetSessionHistory(r.Context(), sessionID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, history)
}

func (s *Server) handleUpdateSession(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")
	userID := r.Context().Value("user_id").(string)

	var req struct {
		Title    *string `json:"title"`
		IsPinned *bool   `json:"is_pinned"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	err := s.webChatService.UpdateSession(r.Context(), sessionID, userID, req.Title, req.IsPinned)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (s *Server) handleDeleteSession(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")
	userID := r.Context().Value("user_id").(string)

	err := s.webChatService.DeleteSession(r.Context(), sessionID, userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
