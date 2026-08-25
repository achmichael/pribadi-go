package rag

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/achmichael/pribadi-go/internal/chunker"
	"github.com/achmichael/pribadi-go/internal/repository"
	"github.com/rs/zerolog"
)

// IngestionService handles document chunking and vector storage
type IngestionService interface {
	IngestText(ctx context.Context, text string, metadata map[string]string) (int, error)
}

type ingestionService struct {
	vectorRepo repository.VectorRepository
	logger     *zerolog.Logger
	batchSize  int // how many chunks per embedding batch (default 5)
}

// NewIngestionService creates a new ingestion service
func NewIngestionService(vectorRepo repository.VectorRepository, logger *zerolog.Logger) IngestionService {
	return &ingestionService{
		vectorRepo: vectorRepo,
		logger:     logger,
		batchSize:  5,
	}
}

// IngestText chunks text using the chunker package, then upserts each chunk
// into the vector store with retry/backoff on embedding calls.
func (s *ingestionService) IngestText(ctx context.Context, text string, metadata map[string]string) (int, error) {
	start := time.Now()
	sourceFile := metadata["source_file"]

	opts := chunker.ChunkOptions{
		MaxTokens: 800,
		Overlap:   100,
	}

	chunks, err := chunker.ChunkText(text, sourceFile, opts)
	if err != nil {
		return 0, fmt.Errorf("chunking failed: %w", err)
	}
	
	if len(chunks) > 0 {
		s.logger.Info().
			Str("source", sourceFile).
			Int("chunk_0_len", len(chunks[0].Content)).
			Msg("[AUDIT] ingestionService chunk 0 length")
	}

	s.logger.Info().
		Str("source", sourceFile).
		Int("total_chunks", len(chunks)).
		Int("text_len", len(text)).
		Dur("chunk_duration", time.Since(start)).
		Msg("Document chunked")

	if len(chunks) == 0 {
		return 0, nil
	}

	// Prepare metadata header string once per document
	var metaHeaderBuilder strings.Builder
	metaHeaderBuilder.WriteString("[Document Metadata]\n")
	for k, v := range metadata {
		metaHeaderBuilder.WriteString(fmt.Sprintf("%s: %s\n", k, v))
	}
	metaHeaderBuilder.WriteString("---\n\n")
	metaHeaderStr := metaHeaderBuilder.String()

	upserted := 0
	for i, chunk := range chunks {
		chunkStart := time.Now()

		chunkMeta := make(map[string]string)
		for k, v := range metadata {
			chunkMeta[k] = v
		}
		chunkMeta["chunk_index"] = fmt.Sprintf("%d", chunk.ChunkIndex)
		chunkMeta["total_chunks"] = fmt.Sprintf("%d", chunk.TotalChunks)

		id := fmt.Sprintf("%s-chunk-%d", sourceFile, chunk.ChunkIndex)

		// Prepend metadata header to chunk content
		enrichedContent := metaHeaderStr + chunk.Content
		
		if i == 0 {
			s.logger.Info().
				Str("source", sourceFile).
				Int("enriched_content_len", len(enrichedContent)).
				Msg("[AUDIT] ingestionService chunk 0 enriched content length")
		}

		err := retryWithBackoff(ctx, 3, func() error {
			return s.vectorRepo.UpsertDocument(ctx, id, enrichedContent, chunkMeta)
		})
		if err != nil {
			s.logger.Error().
				Err(err).
				Int("chunk_index", i).
				Str("source", sourceFile).
				Msg("Failed to upsert chunk after retries")
			// Continue with remaining chunks instead of aborting
			continue
		}

		upserted++

		s.logger.Debug().
			Int("chunk_index", i).
			Int("chunk_tokens", estimateTokens(enrichedContent)).
			Dur("embed_duration", time.Since(chunkStart)).
			Msg("Chunk upserted")
	}

	s.logger.Info().
		Str("source", sourceFile).
		Int("upserted", upserted).
		Int("total_chunks", len(chunks)).
		Dur("total_duration", time.Since(start)).
		Msg("Ingestion complete")

	return upserted, nil
}

// estimateTokens mirrors chunker logic: chars/4.
func estimateTokens(s string) int {
	n := 0
	for range s {
		n++
	}
	return (n + 3) / 4
}

// retryWithBackoff retries fn up to maxAttempts with exponential backoff.
func retryWithBackoff(ctx context.Context, maxAttempts int, fn func() error) error {
	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		lastErr = fn()
		if lastErr == nil {
			return nil
		}

		if attempt < maxAttempts-1 {
			backoff := time.Duration(math.Pow(2, float64(attempt))) * time.Second
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
		}
	}
	return fmt.Errorf("after %d attempts: %w", maxAttempts, lastErr)
}
