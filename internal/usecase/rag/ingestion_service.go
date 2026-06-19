package rag

import (
	"context"
	"fmt"
	"strings"

	"github.com/achmichael/pribadi-go/internal/repository"
)

// IngestionService handles document chunking and vector storage
type IngestionService interface {
	IngestText(ctx context.Context, text string, metadata map[string]string) (int, error)
}

type ingestionService struct {
	vectorRepo repository.VectorRepository
}

// NewIngestionService creates a new ingestion service
func NewIngestionService(vectorRepo repository.VectorRepository) IngestionService {
	return &ingestionService{
		vectorRepo: vectorRepo,
	}
}

// IngestText chunks text and upserts into vector DB
func (s *ingestionService) IngestText(ctx context.Context, text string, metadata map[string]string) (int, error) {
	chunks := s.chunkText(text, 500, 50)

	for i, chunk := range chunks {
		chunkMeta := make(map[string]string)
		for k, v := range metadata {
			chunkMeta[k] = v
		}
		chunkMeta["chunk_index"] = fmt.Sprintf("%d", i)

		id := fmt.Sprintf("%s-%d", metadata["source_file"], i)
		err := s.vectorRepo.UpsertDocument(ctx, id, chunk, chunkMeta)
		if err != nil {
			return i, fmt.Errorf("failed to upsert chunk %d: %w", i, err)
		}
	}

	return len(chunks), nil
}

// chunkText implements simple word-count based sliding window chunker
func (s *ingestionService) chunkText(text string, chunkSize, overlap int) []string {
	words := strings.Fields(text)
	if len(words) <= chunkSize {
		return []string{text}
	}

	var chunks []string
	step := chunkSize - overlap
	if step <= 0 {
		step = 1
	}

	for i := 0; i < len(words); i += step {
		end := i + chunkSize
		if end > len(words) {
			end = len(words)
		}
		
		chunk := strings.Join(words[i:end], " ")
		chunks = append(chunks, chunk)

		if end == len(words) {
			break
		}
	}

	return chunks
}
