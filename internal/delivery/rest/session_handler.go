package rest

import (
	"encoding/json"
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
		Title string `json:"title"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	session, err := s.webChatService.CreateSession(r.Context(), userID, req.Title)
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
