package rest

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"io"
)

func (s *Server) handleFileUpload(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(32 << 20)
	if err != nil {
		respondError(w, http.StatusBadRequest, "File too large")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid file")
		return
	}
	defer file.Close()

	userID := r.Context().Value("user_id").(string)

	uploadDir := filepath.Join("data", "uploads", userID)
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	origName := filepath.Base(header.Filename)
	origName = strings.ReplaceAll(origName, "..", "")
	origName = strings.ReplaceAll(origName, "/", "")
	origName = strings.ReplaceAll(origName, "\\", "")

	ext := filepath.Ext(origName)
	diskName := uuid.New().String() + ext
	savePath := filepath.Join(uploadDir, diskName)

	dst, err := os.Create(savePath)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to save file")
		return
	}

	mimeType := header.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	job, err := s.webChatService.CreateUploadJob(r.Context(), userID, origName, savePath, mimeType, header.Size)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create upload job")
		return
	}

	s.webChatService.ProcessUploadJobAsync(job.ID)

	respondJSON(w, http.StatusAccepted, job)
}
