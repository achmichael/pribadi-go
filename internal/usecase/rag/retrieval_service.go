package rag

import (
	"context"
	"strings"
	"time"

	"github.com/achmichael/pribadi-go/internal/repository"
	"github.com/rs/zerolog"
)

// PromptContext contains retrieval results for LLM prompt
type PromptContext struct {
	Context    string
	Sources    []string
	HasResults bool
}

// RetrievalService handles semantic search
type RetrievalService interface {
	Retrieve(ctx context.Context, question string) (PromptContext, error)
}

type retrievalService struct {
	vectorRepo repository.VectorRepository
	logger     *zerolog.Logger
}

// NewRetrievalService creates a new retrieval service
func NewRetrievalService(vectorRepo repository.VectorRepository, logger *zerolog.Logger) RetrievalService {
	return &retrievalService{
		vectorRepo: vectorRepo,
		logger:     logger,
	}
}

// Retrieve searches top relevant chunks
func (s *retrievalService) Retrieve(ctx context.Context, question string) (PromptContext, error) {
	start := time.Now()

	s.logger.Info().
		Int("query_len", len(question)).
		Msg("[retrieval] starting search")

	results, err := s.vectorRepo.Search(ctx, question, 8)
	searchDur := time.Since(start)

	if err != nil {
		s.logger.Error().Err(err).
			Dur("duration_ms", searchDur).
			Msg("[retrieval] search failed")
		return PromptContext{}, err
	}

	if len(results) == 0 {
		s.logger.Info().
			Dur("duration_ms", searchDur).
			Msg("[retrieval] no results found")
		return PromptContext{HasResults: false}, nil
	}

	var sb strings.Builder
	var sources []string

	// Only include results above relevance threshold.
	const minScore float32 = 0.3
	for i, res := range results {
		if res.Score < minScore {
			continue
		}
		if i > 0 {
			sb.WriteString("\n---\n")
		}
		sb.WriteString(res.Content)

		source := res.Metadata["source_file"]
		if source != "" {
			sources = append(sources, source)
		}
	}

	contextText := sb.String()

	s.logger.Info().
		Int("results_count", len(results)).
		Int("context_len", len(contextText)).
		Float32("top_score", results[0].Score).
		Dur("duration_ms", searchDur).
		Msg("[retrieval] search done")

	return PromptContext{
		Context:    contextText,
		Sources:    sources,
		HasResults: true,
	}, nil
}
