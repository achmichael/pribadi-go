package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

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
	DeleteSession(ctx context.Context, sessionID, userID string) error

	// Messages & Streaming
	GetSessionHistory(ctx context.Context, sessionID string) ([]domain.WebChatMessage, error)
	GenerateChatTitle(ctx context.Context, message string) (string, error)
	StreamChat(ctx context.Context, userID, sessionID, message, model string) (<-chan ollama.StreamChunk, error)

	// API Keys
	SaveAPIKey(ctx context.Context, userID, provider, key string) error
	GetAPIKey(ctx context.Context, userID, provider string) (string, error)

	// Uploads for RAG
	CreateUploadJob(ctx context.Context, userID, fileName, filePath, mimeType string, fileSize int64) (*domain.WebUploadJob, error)
	ProcessUploadJobAsync(jobID string)
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

func (s *webChatService) DeleteSession(ctx context.Context, sessionID, userID string) error {
	return s.repo.DeleteSession(ctx, sessionID, userID)
}

// ─── Messages & Streaming ─────────────────────────────────────────────────

func (s *webChatService) GetSessionHistory(ctx context.Context, sessionID string) ([]domain.WebChatMessage, error) {
	return s.repo.ListMessages(ctx, sessionID)
}

func (s *webChatService) StreamChat(ctx context.Context, userID, sessionID, message, model string) (<-chan ollama.StreamChunk, error) {
	// 1. Verify session belongs to user
	session, err := s.repo.GetSession(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}
	if session == nil || session.UserID != userID {
		return nil, errors.New("session not found or unauthorized")
	}

	// 2. Save User Message
	userMsg := domain.WebChatMessage{
		ID:        uuid.New().String(),
		SessionID: sessionID,
		Role:      "user",
		Content:   message,
		Model:     model,
		CreatedAt: time.Now(),
	}
	_ = s.repo.CreateMessage(ctx, userMsg)

	// 3. Build Chat History for Ollama
	history, _ := s.repo.ListMessages(ctx, sessionID)
	var oMessages []ollama.ChatMessage

	// Prepend System Prompt if any (TODO: fetch from AgentConfig)
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

	// 4. Retrieve RAG Context (Optional, simplified for now)
	// We search user documents using the full message as question, targetDocID = "" for all docs
	pCtx, err := s.ragRetrieve.Retrieve(ctx, message, "")
	if err == nil && pCtx.HasResults {
		contextText := "Context from user documents:\n" + pCtx.Context
		// Append to the last message (user message)
		lastIdx := len(oMessages) - 1
		oMessages[lastIdx].Content = fmt.Sprintf("%s\n\n%s", oMessages[lastIdx].Content, contextText)
	}

	// 5. Stream from Ollama
	// TODO: Support BYOK models by routing to OpenAI/Anthropic SDKs here if model != "local"
	stream, err := s.ollamaClient.ChatStream(ctx, oMessages, nil)
	if err != nil {
		return nil, err
	}

	// 6. Intercept stream to save final assistant message
	interceptedStream := make(chan ollama.StreamChunk)
	go func() {
		defer close(interceptedStream)

		var fullContent string
		var toolCalls []ollama.ToolCall

		for chunk := range stream {
			if chunk.Err == nil && !chunk.Done {
				fullContent += chunk.Content
				if len(chunk.ToolCalls) > 0 {
					toolCalls = append(toolCalls, chunk.ToolCalls...)
				}
			}
			interceptedStream <- chunk
		}

		// Stream finished, save assistant message
		tcJSON, _ := json.Marshal(toolCalls)
		asstMsg := domain.WebChatMessage{
			ID:            uuid.New().String(),
			SessionID:     sessionID,
			Role:          "assistant",
			Content:       fullContent,
			Model:         model,
			ToolCallsJSON: string(tcJSON),
			CreatedAt:     time.Now(),
		}
		_ = s.repo.CreateMessage(context.Background(), asstMsg)
	}()

	return interceptedStream, nil
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

		// 3. Ingest Document
		chunks, err := s.ragIngest.IngestText(ctx, text, map[string]string{
			"source": "web_dashboard",
			"source_file": job.FileName,
			"mime_type": job.MimeType,
			"user_id": job.UserID,
		})

		// 4. Mark completed/failed
		if err != nil {
			s.logger.Error().Err(err).Str("job_id", jobID).Msg("RAG ingestion failed")
			_ = s.repo.UpdateUploadJobStatus(ctx, jobID, "failed", err.Error(), "")
			return
		}

		docID := uuid.New().String() // Generating docID here since IngestText returns chunk count
		s.logger.Info().Str("job_id", jobID).Int("chunks", chunks).Msg("RAG ingestion completed")
		_ = s.repo.UpdateUploadJobStatus(ctx, jobID, "completed", "", docID)
	}()
}
