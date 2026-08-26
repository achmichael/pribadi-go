package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/achmichael/pribadi-go/internal/budget"
	"github.com/achmichael/pribadi-go/internal/domain"
	"github.com/achmichael/pribadi-go/internal/repository"
	"github.com/achmichael/pribadi-go/internal/usecase/rag"
	"github.com/achmichael/pribadi-go/pkg/crypto"
	"github.com/achmichael/pribadi-go/pkg/ollama"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type WebChatService interface {
	// Sessions
	CreateSession(ctx context.Context, userID, title string) (*domain.WebChatSession, error)
	GetSession(ctx context.Context, sessionID string) (*domain.WebChatSession, error)
	ListSessions(ctx context.Context, userID string) ([]domain.WebChatSession, error)
	UpdateSession(ctx context.Context, sessionID, userID string, title *string, isPinned *bool) error
	DeleteSession(ctx context.Context, sessionID, userID string) error

	// Messages & Streaming
	GetSessionHistory(ctx context.Context, sessionID string) ([]domain.WebChatMessage, error)
	GenerateChatTitle(ctx context.Context, message string) (string, error)
	StreamChat(ctx context.Context, userID, sessionID, message, model, fileJobID string) (<-chan ollama.StreamChunk, error)

	// API Keys
	SaveAPIKey(ctx context.Context, userID, provider, key string) error
	GetAPIKey(ctx context.Context, userID, provider string) (string, error)

	// Uploads for RAG
	CreateUploadJob(ctx context.Context, userID, fileName, filePath, mimeType string, fileSize int64) (*domain.WebUploadJob, error)
	ProcessUploadJobAsync(jobID string)
	ListUploadJobs(ctx context.Context, userID string) ([]domain.WebUploadJob, error)
	DeleteUploadJob(ctx context.Context, userID, jobID string) error
	PurgeUserStorage(ctx context.Context, userID string) error
}

type webChatService struct {
	repo          repository.WebChatRepository
	ollamaClient  *ollama.OllamaClient
	ragIngest     rag.IngestionService
	ragRetrieve   rag.RetrievalService
	extraction    ExtractionService
	encryptionKey []byte
	logger        *zerolog.Logger
}

func NewWebChatService(
	repo repository.WebChatRepository,
	ollamaClient *ollama.OllamaClient,
	ragIngest rag.IngestionService,
	ragRetrieve rag.RetrievalService,
	extraction ExtractionService,
	encryptionKey string,
	logger *zerolog.Logger,
) WebChatService {
	// Ensure key is exactly 32 bytes for AES-GCM
	keyBytes := []byte(encryptionKey)
	if len(keyBytes) > 32 {
		keyBytes = keyBytes[:32]
	} else if len(keyBytes) < 32 {
		padded := make([]byte, 32)
		copy(padded, keyBytes)
		keyBytes = padded
	}

	return &webChatService{
		repo:          repo,
		ollamaClient:  ollamaClient,
		ragIngest:     ragIngest,
		ragRetrieve:   ragRetrieve,
		extraction:    extraction,
		encryptionKey: keyBytes,
		logger:        logger,
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (s *webChatService) GenerateChatTitle(ctx context.Context, message string) (string, error) {
	response, err := s.ollamaClient.GenerateChatTitle(ctx, message)

	if err != nil {
		return "", err
	}

	return strings.TrimSpace(response), nil
}

// ─── Sessions ─────────────────────────────────────────────────────────────

func (s *webChatService) CreateSession(ctx context.Context, userID, title string) (*domain.WebChatSession, error) {
	if title == "" {
		title = "New Chat"
	}
	session := domain.WebChatSession{
		ID:        uuid.New().String(),
		UserID:    userID,
		Title:     title,
		IsPinned:  false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := s.repo.CreateSession(ctx, session); err != nil {
		return nil, err
	}
	return &session, nil
}

func (s *webChatService) GetSession(ctx context.Context, sessionID string) (*domain.WebChatSession, error) {
	return s.repo.GetSession(ctx, sessionID)
}

func (s *webChatService) ListSessions(ctx context.Context, userID string) ([]domain.WebChatSession, error) {
	return s.repo.ListSessions(ctx, userID)
}

func (s *webChatService) UpdateSession(ctx context.Context, sessionID, userID string, title *string, isPinned *bool) error {
	session, err := s.repo.GetSession(ctx, sessionID)
	if err != nil {
		return err
	}
	if session == nil || session.UserID != userID {
		return errors.New("session not found or unauthorized")
	}

	if title != nil {
		session.Title = *title
	}
	if isPinned != nil {
		session.IsPinned = *isPinned
	}

	return s.repo.UpdateSession(ctx, *session)
}

func (s *webChatService) DeleteSession(ctx context.Context, sessionID, userID string) error {
	return s.repo.DeleteSession(ctx, sessionID, userID)
}

// ─── Messages & Streaming ─────────────────────────────────────────────────

func (s *webChatService) GetSessionHistory(ctx context.Context, sessionID string) ([]domain.WebChatMessage, error) {
	return s.repo.ListMessages(ctx, sessionID)
}

func (s *webChatService) StreamChat(ctx context.Context, userID, sessionID, message, model, fileJobID string) (<-chan ollama.StreamChunk, error) {
	session, err := s.repo.GetSession(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}
	if session == nil || session.UserID != userID {
		return nil, errors.New("session not found or unauthorized")
	}

	userMsg := domain.WebChatMessage{
		ID:        uuid.New().String(),
		SessionID: sessionID,
		Role:      "user",
		Content:   message,
		Model:     model,
		FileJobID: fileJobID,
		CreatedAt: time.Now(),
	}
	_ = s.repo.CreateMessage(ctx, userMsg)

	history, _ := s.repo.ListMessages(ctx, sessionID)
	var oMessages []ollama.ChatMessage

	oMessages = append(oMessages, ollama.ChatMessage{
		Role:    "system",
		Content: "You are pribadi-go, a helpful AI assistant. Always respond in Markdown.",
	})

	for _, m := range history {
		oMessages = append(oMessages, ollama.ChatMessage{
			Role:    m.Role,
			Content: m.Content,
		})
	}

	interceptedStream := make(chan ollama.StreamChunk, 64)

	go func() {
		defer close(interceptedStream)

		s.logger.Info().
			Str("fileJobID", fileJobID).
			Str("message", message).
			Msg("[AUDIT] WebChatService StreamChat started processing")

		fileFullInjected := false
		targetDocID := ""

		if fileJobID != "" {
			job, _ := s.repo.GetUploadJob(ctx, fileJobID)
			if job != nil {
				targetDocID = job.DocumentID
			}

			interceptedStream <- ollama.StreamChunk{
				Stage: &ollama.StageEvent{Stage: "extracting_file", Message: "Processing attached file..."},
			}

			fileContent := s.waitAndExtractFile(ctx, fileJobID)

			s.logger.Info().
				Str("fileJobID", fileJobID).
				Int("fileContentLength", len(fileContent)).
				Msg("[AUDIT] WebChatService extracted file content length")

			if ctx.Err() != nil {
				return
			}

			if strings.TrimSpace(fileContent) == "" {
				interceptedStream <- ollama.StreamChunk{
					Stage: &ollama.StageEvent{Stage: "file_extracted", Message: "File extraction returned empty content. The file may be unreadable."},
				}
			} else {
				fileTokens := budget.EstimateTokens(fileContent)

				var historyTexts []string
				for _, m := range oMessages {
					historyTexts = append(historyTexts, m.Content)
				}
				calc := budget.NewCalculator(s.ollamaClient.NumCtx(), s.ollamaClient.NumPredict())

				if calc.FitsInContext(fileTokens, oMessages[0].Content, historyTexts[1:]) {
					s.logger.Info().
						Int("file_tokens", fileTokens).
						Msg("[routing] full-inject path: file fits in context budget")

					lastIdx := len(oMessages) - 1
					userText := oMessages[lastIdx].Content

					if strings.TrimSpace(userText) == "" {
						oMessages[lastIdx].Content = fmt.Sprintf("[FILE UPLOAD — Isi lengkap file yang baru diupload user]\n\n---\n%s\n---\n\nAnalisis dan rangkum isi dokumen di atas.", fileContent)
					} else {
						oMessages[lastIdx].Content = fmt.Sprintf("[FILE UPLOAD — Isi lengkap file yang baru diupload user]\n\nInstruksi user: %s\n\n---\n%s\n---\n\nIkuti instruksi user di atas. Konten file disediakan sebagai konteks.", userText, fileContent)
					}

					fileFullInjected = true
					interceptedStream <- ollama.StreamChunk{
						Stage: &ollama.StageEvent{Stage: "file_extracted", Message: "File content injected directly (fits in context)"},
					}
				} else {
					s.logger.Info().
						Int("file_tokens", fileTokens).
						Msg("[routing] RAG path: file too large for context, using chunked retrieval")

					interceptedStream <- ollama.StreamChunk{
						Stage: &ollama.StageEvent{Stage: "file_extracted", Message: "File too large for direct injection, using document search"},
					}
				}
			}
		}

		if !fileFullInjected {
			interceptedStream <- ollama.StreamChunk{
				Stage: &ollama.StageEvent{Stage: "retrieving_context", Tool: "qdrant_search", Message: "Searching documents..."},
			}

			var pCtx rag.PromptContext
			var ragErr error

			if targetDocID != "" {
				pCtx, ragErr = s.ragRetrieve.RetrieveFiltered(ctx, message, map[string]string{
					"document_id": targetDocID,
				})
				s.logger.Info().
					Str("targetDocID", targetDocID).
					Bool("has_results", pCtx.HasResults).
					Msg("[routing] RAG filtered search by document_id")
			} else {
				// Enforce session_id filter for general queries in the session
				pCtx, ragErr = s.ragRetrieve.RetrieveFiltered(ctx, message, map[string]string{
					"session_id": sessionID,
				})
				s.logger.Info().
					Str("session_id", sessionID).
					Bool("has_results", pCtx.HasResults).
					Msg("[routing] RAG filtered search by session_id")
			}

			if ctx.Err() != nil {
				return
			}

			if ragErr == nil && pCtx.HasResults {
				lastIdx := len(oMessages) - 1
				if targetDocID != "" {
					oMessages[lastIdx].Content = fmt.Sprintf("%s\n\n[FILE UPLOAD — Potongan relevan dari file yang baru diupload user]\n%s", oMessages[lastIdx].Content, pCtx.Context)
				} else {
					oMessages[lastIdx].Content = fmt.Sprintf("%s\n\nContext from user documents:\n%s", oMessages[lastIdx].Content, pCtx.Context)
				}

				interceptedStream <- ollama.StreamChunk{
					Stage: &ollama.StageEvent{Stage: "context_retrieved", Tool: "qdrant_search", Message: fmt.Sprintf("Found %d relevant chunks", len(strings.Split(pCtx.Context, "\n\n")))},
				}
			} else {
				interceptedStream <- ollama.StreamChunk{
					Stage: &ollama.StageEvent{Stage: "context_retrieved", Message: "No relevant documents found"},
				}
			}
		}

		if ctx.Err() != nil {
			return
		}

		interceptedStream <- ollama.StreamChunk{
			Stage: &ollama.StageEvent{Stage: "generating", Message: "Generating response..."},
		}

		stream, streamErr := s.ollamaClient.ChatStreamWithThink(ctx, oMessages, nil, true)
		if streamErr != nil {
			interceptedStream <- ollama.StreamChunk{Err: streamErr}
			return
		}

		var fullContent string
		var toolCalls []ollama.ToolCall
		interrupted := false

		for chunk := range stream {
			select {
			case <-ctx.Done():
				interrupted = true
				goto save
			default:
			}

			if chunk.Err == nil && !chunk.Done {
				fullContent += chunk.Content
				if len(chunk.ToolCalls) > 0 {
					toolCalls = append(toolCalls, chunk.ToolCalls...)
				}
			}
			interceptedStream <- chunk
		}

	save:
		tcJSON, _ := json.Marshal(toolCalls)
		content := fullContent
		if interrupted {
			content += "\n\n---\n*Generation stopped by user*"
		}
		asstMsg := domain.WebChatMessage{
			ID:            uuid.New().String(),
			SessionID:     sessionID,
			Role:          "assistant",
			Content:       content,
			Model:         model,
			ToolCallsJSON: string(tcJSON),
			CreatedAt:     time.Now(),
		}
		_ = s.repo.CreateMessage(context.Background(), asstMsg)
	}()

	return interceptedStream, nil
}

func (s *webChatService) waitAndExtractFile(ctx context.Context, jobID string) string {
	for i := 0; i < 30; i++ {
		if ctx.Err() != nil {
			return ""
		}

		job, err := s.repo.GetUploadJob(ctx, jobID)
		if err != nil || job == nil {
			return ""
		}

		if job.Status == "completed" || job.Status == "failed" {
			if job.Status == "failed" {
				s.logger.Warn().Str("job_id", jobID).Msg("File extraction failed")
				return ""
			}
			fileBytes, err := os.ReadFile(job.FilePath)
			if err != nil {
				s.logger.Warn().Err(err).Str("job_id", jobID).Msg("Failed to read file for inline extraction")
				return ""
			}
			text, err := s.extraction.ExtractText(ctx, fileBytes, job.MimeType)
			if err != nil {
				s.logger.Warn().Err(err).Str("job_id", jobID).Msg("Failed to extract text for inline use")
				return ""
			}
			return text
		}

		time.Sleep(500 * time.Millisecond)
	}

	s.logger.Warn().Str("job_id", jobID).Msg("Timed out waiting for file processing")
	return ""
}

// ─── API Keys ─────────────────────────────────────────────────────────────

func (s *webChatService) SaveAPIKey(ctx context.Context, userID, provider, key string) error {
	if key == "" {
		return s.repo.DeleteAPIKey(ctx, userID, provider)
	}

	encrypted, err := crypto.Encrypt([]byte(key), s.encryptionKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt key: %w", err)
	}

	apiKey := domain.UserAPIKey{
		UserID:       userID,
		Provider:     provider,
		EncryptedKey: encrypted,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	return s.repo.UpsertAPIKey(ctx, apiKey)
}

func (s *webChatService) GetAPIKey(ctx context.Context, userID, provider string) (string, error) {
	key, err := s.repo.GetAPIKey(ctx, userID, provider)
	if err != nil {
		return "", err
	}
	if key == nil {
		return "", nil // Not found
	}

	decrypted, err := crypto.Decrypt(key.EncryptedKey, s.encryptionKey)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt key: %w", err)
	}

	return string(decrypted), nil
}

// ─── Uploads for RAG ──────────────────────────────────────────────────────

func (s *webChatService) CreateUploadJob(ctx context.Context, userID, fileName, filePath, mimeType string, fileSize int64) (*domain.WebUploadJob, error) {
	job := domain.WebUploadJob{
		ID:        uuid.New().String(),
		UserID:    userID,
		FileName:  fileName,
		FilePath:  filePath,
		FileSize:  fileSize,
		MimeType:  mimeType,
		Status:    "pending",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := s.repo.CreateUploadJob(ctx, job); err != nil {
		return nil, err
	}
	return &job, nil
}

func (s *webChatService) ProcessUploadJobAsync(jobID string) {
	go func() {
		ctx := context.Background()

		// 1. Mark processing
		_ = s.repo.UpdateUploadJobStatus(ctx, jobID, "processing", "", "")

		job, err := s.repo.GetUploadJob(ctx, jobID)
		if err != nil || job == nil {
			s.logger.Error().Err(err).Str("job_id", jobID).Msg("Failed to get upload job")
			return
		}

		s.logger.Info().Str("job_id", jobID).Str("file", job.FileName).Msg("Processing RAG upload")

		// 2. Extract Text from file
		fileBytes, err := os.ReadFile(job.FilePath)
		if err != nil {
			s.logger.Error().Err(err).Str("job_id", jobID).Msg("Failed to read file for extraction")
			_ = s.repo.UpdateUploadJobStatus(ctx, jobID, "failed", err.Error(), "")
			return
		}

		text, err := s.extraction.ExtractText(ctx, fileBytes, job.MimeType)
		if err != nil {
			s.logger.Error().Err(err).Str("job_id", jobID).Msg("Failed to extract text from document")
			_ = s.repo.UpdateUploadJobStatus(ctx, jobID, "failed", err.Error(), "")
			return
		}

		s.logger.Info().
			Str("job_id", jobID).
			Int("textLength", len(text)).
			Msg("[AUDIT] WebChatService extracted raw text from file")

		docID := uuid.New().String() // Generating docID here since IngestText returns chunk count

		s.logger.Info().
			Str("job_id", jobID).
			Str("file_name", job.FileName).
			Str("document_id", docID).
			Str("user_id", job.UserID).
			Msg("[AUDIT] WebChatService ProcessUploadJobAsync preparing to ingest text")

		// 3. Ingest Document
		chunks, err := s.ragIngest.IngestText(ctx, text, map[string]string{
			"source":      "web_dashboard",
			"source_file": job.FileName,
			"mime_type":   job.MimeType,
			"user_id":     job.UserID,
			"document_id": docID,
			"filename":    job.FileName,
			"uploaded_at": time.Now().UTC().Format(time.RFC3339),
		})

		// 4. Mark completed/failed
		if err != nil {
			s.logger.Error().Err(err).Str("job_id", jobID).Msg("RAG ingestion failed")
			_ = s.repo.UpdateUploadJobStatus(ctx, jobID, "failed", err.Error(), "")
			return
		}

		s.logger.Info().Str("job_id", jobID).Int("chunks", chunks).Str("document_id", docID).Msg("[AUDIT] WebChatService RAG ingestion completed successfully")
		_ = s.repo.UpdateUploadJobStatus(ctx, jobID, "completed", "", docID)
	}()
}

func (s *webChatService) ListUploadJobs(ctx context.Context, userID string) ([]domain.WebUploadJob, error) {
	return s.repo.ListUploadJobs(ctx, userID)
}

func (s *webChatService) DeleteUploadJob(ctx context.Context, userID, jobID string) error {
	job, err := s.repo.GetUploadJob(ctx, jobID)
	if err != nil {
		return err
	}
	if job == nil || job.UserID != userID {
		return fmt.Errorf("job not found or unauthorized")
	}

	// Delete from local disk
	if job.FilePath != "" {
		_ = os.Remove(job.FilePath)
	}

	// Delete from Qdrant if document_id exists
	if job.DocumentID != "" {
		_ = s.ragIngest.DeleteDocument(ctx, job.DocumentID)
	}

	return s.repo.DeleteUploadJob(ctx, jobID)
}

func (s *webChatService) PurgeUserStorage(ctx context.Context, userID string) error {
	jobs, err := s.repo.ListUploadJobs(ctx, userID)
	if err != nil {
		return err
	}
	
	// Delete all jobs files and from DB
	for _, job := range jobs {
		if job.FilePath != "" {
			_ = os.Remove(job.FilePath)
		}
		_ = s.repo.DeleteUploadJob(ctx, job.ID)
	}

	// Delete EVERYTHING in Qdrant for this user
	_ = s.ragIngest.PurgeUserDocuments(ctx, userID)

	return nil
}
