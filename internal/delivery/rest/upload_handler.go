package rest

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
)

// handleFileUpload handles multipart form data upload and triggers RAG ingestion
func (s *Server) handleFileUpload(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(32 << 20) // 32MB max memory
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

	// Ensure temp dir exists
	tempDir := "data/temp_uploads"
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	savePath := filepath.Join(tempDir, header.Filename) // Using original name for now, should UUID in prod

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

	job, err := s.webChatService.CreateUploadJob(r.Context(), userID, header.Filename, savePath, mimeType, header.Size)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create upload job")
		return
	}

	// Trigger async processing
	s.webChatService.ProcessUploadJobAsync(job.ID)

	respondJSON(w, http.StatusAccepted, job)
}
