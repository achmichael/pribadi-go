package rest

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/achmichael/pribadi-go/internal/domain"
	"github.com/achmichael/pribadi-go/internal/orchestrator"
	"github.com/achmichael/pribadi-go/internal/repository"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type ExtractionService interface {
	ExtractText(ctx context.Context, fileBytes []byte, mimeType string) (string, error)
}

type WebInboundRawEvent struct {
	UserID    string
	SessionID string
	Message   string
	FileJobID string
	Model     string
}

type webInboundAdapter struct {
	repo       repository.WebChatRepository
	extraction ExtractionService
	logger     *zerolog.Logger
}

func NewInboundAdapter(
	repo repository.WebChatRepository,
	extraction ExtractionService,
	logger *zerolog.Logger,
) orchestrator.InboundAdapter {
	return &webInboundAdapter{
		repo:       repo,
		extraction: extraction,
		logger:     logger,
	}
}

func (a *webInboundAdapter) Normalize(ctx context.Context, rawEvent any) (*orchestrator.NormalizedInboundMessage, error) {
	req, ok := rawEvent.(WebInboundRawEvent)
	if !ok {
		return nil, fmt.Errorf("expected WebInboundRawEvent, got %T", rawEvent)
	}

	session, err := a.repo.GetSession(ctx, req.SessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}
	if session == nil || session.UserID != req.UserID {
		return nil, fmt.Errorf("session not found or unauthorized")
	}

	normalized := &orchestrator.NormalizedInboundMessage{
		UserID:     req.UserID,
		PlatformID: "web",
		SessionID:  req.SessionID,
		MessageID:  uuid.New().String(),
		Text:       req.Message,
		Metadata:   map[string]any{"model": req.Model},
		Timestamp:  time.Now(),
	}

	if req.FileJobID != "" {
		text, fileName, mimeType := a.waitAndExtractFile(ctx, req.FileJobID)
		if text != "" {
			normalized.Attachments = append(normalized.Attachments, orchestrator.Attachment{
				Type:          orchestrator.AttachmentTypeDocument,
				FileName:      fileName,
				MimeType:      mimeType,
				ExtractedText: text,
			})
		}
	}

	userMsg := domain.WebChatMessage{
		ID:        normalized.MessageID,
		SessionID: req.SessionID,
		Role:      "user",
		Content:   req.Message,
		Model:     req.Model,
		FileJobID: req.FileJobID,
		CreatedAt: normalized.Timestamp,
	}
	_ = a.repo.CreateMessage(ctx, userMsg)

	return normalized, nil
}

func (a *webInboundAdapter) waitAndExtractFile(ctx context.Context, jobID string) (string, string, string) {
	for i := 0; i < 30; i++ {
		if ctx.Err() != nil {
			return "", "", ""
		}

		job, err := a.repo.GetUploadJob(ctx, jobID)
		if err != nil || job == nil {
			return "", "", ""
		}

		if job.Status == "completed" || job.Status == "failed" {
			if job.Status == "failed" {
				a.logger.Warn().Str("job_id", jobID).Msg("File extraction failed")
				return "", "", ""
			}
			fileBytes, err := os.ReadFile(job.FilePath)
			if err != nil {
				a.logger.Warn().Err(err).Str("job_id", jobID).Msg("Failed to read file for inline extraction")
				return "", "", ""
			}
			text, err := a.extraction.ExtractText(ctx, fileBytes, job.MimeType)
			if err != nil {
				a.logger.Warn().Err(err).Str("job_id", jobID).Msg("Failed to extract text for inline use")
				return "", "", ""
			}
			return text, job.FileName, job.MimeType
		}

		time.Sleep(500 * time.Millisecond)
	}

	a.logger.Warn().Str("job_id", jobID).Msg("Timed out waiting for file processing")
	return "", "", ""
}

type webOutboundAdapter struct {
	streamCh chan orchestrator.ResponseChunk
	logger   *zerolog.Logger
}

func NewOutboundAdapter(logger *zerolog.Logger) (orchestrator.OutboundAdapter, chan orchestrator.ResponseChunk) {
	ch := make(chan orchestrator.ResponseChunk, 64)
	return &webOutboundAdapter{
		streamCh: ch,
		logger:   logger,
	}, ch
}

func (a *webOutboundAdapter) Deliver(ctx context.Context, sessionID string, chunk orchestrator.ResponseChunk) error {
	// Recover dari panic send on closed channel (just in case)
	defer func() {
		if r := recover(); r != nil {
			a.logger.Warn().Interface("recover", r).Msg("Recovered panic in Deliver (channel likely closed)")
		}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case a.streamCh <- chunk:
		return nil
	}
}
