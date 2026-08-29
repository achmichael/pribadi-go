package rag

import (
	"context"
	"strings"
	"time"
	"fmt"

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
	Retrieve(ctx context.Context, question string, targetDocID string) (PromptContext, error)
	RetrieveFiltered(ctx context.Context, question string, filters map[string]string) (PromptContext, error)
	PurgeRAGDocuments(ctx context.Context, userID string) error
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
func (s *retrievalService) Retrieve(ctx context.Context, question string, targetDocID string) (PromptContext, error) {
	start := time.Now()

	s.logger.Info().
		Int("query_len", len(question)).
		Str("target_doc_id", targetDocID).
		Msg("[retrieval] starting search")

	results, err := s.vectorRepo.Search(ctx, question, 4, targetDocID)
	searchDur := time.Since(start)

	for _, res := range results {
		fmt.Print("res", res.Content)
	}
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
	var metadataHeader string
	headerWritten := false

	// Only include results above relevance threshold.
	const minScore float32 = 0.3
	for i, res := range results {
		if res.Score < minScore {
			continue
		}
		
		content := res.Content
		
		// If we haven't extracted the header yet and we see it, extract it
		if strings.HasPrefix(content, "[Document Metadata]\n") {
			parts := strings.SplitN(content, "---\n\n", 2)
			if len(parts) == 2 {
				if metadataHeader == "" {
					metadataHeader = parts[0] + "---\n\n"
				}
				content = parts[1] // Use only the body for the chunk
			}
		}

		// Write header only once at the very beginning of the context
		if !headerWritten && metadataHeader != "" {
			sb.WriteString(metadataHeader)
			headerWritten = true
		} else if i > 0 && metadataHeader != "" {
			// Only add separator if we're not the first chunk and header is already written
			sb.WriteString("\n---\n")
		} else if i > 0 {
			sb.WriteString("\n---\n")
		}
		
		sb.WriteString(content)

		source := res.Metadata["source_file"]
		if source != "" {
			sources = append(sources, source)
		}
	}

	contextText := sb.String()

	s.logger.Info().
		Int("results_count", len(results)).
		Int("context_len", len(contextText)).
		Str("context_content", contextText). // <-- LOG CONTENT RAG
		Float32("top_score", results[0].Score).
		Dur("duration_ms", searchDur).
		Msg("[retrieval] search done")

	return PromptContext{
		Context:    contextText,
		Sources:    sources,
		HasResults: true,
	}, nil
}

func (s *retrievalService) PurgeRAGDocuments(ctx context.Context, userID string) error {
	return s.vectorRepo.PurgeRAGDocuments(ctx, userID)
}

func (s *retrievalService) RetrieveFiltered(ctx context.Context, question string, filters map[string]string) (PromptContext, error) {
	start := time.Now()

	s.logger.Info().
		Int("query_len", len(question)).
		Int("filter_count", len(filters)).
		Msg("[retrieval] starting filtered search")

	results, err := s.vectorRepo.SearchWithFilters(ctx, question, 4, filters)
	if err != nil {
		return PromptContext{}, err
	}

	if len(results) == 0 {
		s.logger.Info().Dur("ms", time.Since(start)).Msg("[retrieval] filtered search: no results")
		return PromptContext{HasResults: false}, nil
	}

	var sb strings.Builder
	var sources []string
	var metadataHeader string
	headerWritten := false

	const minScore float32 = 0.3
	for i, res := range results {
		if res.Score < minScore {
			continue
		}

		content := res.Content
		
		if strings.HasPrefix(content, "[Document Metadata]\n") {
			parts := strings.SplitN(content, "---\n\n", 2)
			if len(parts) == 2 {
				if metadataHeader == "" {
					metadataHeader = parts[0] + "---\n\n"
				}
				content = parts[1]
			}
		}

		if !headerWritten && metadataHeader != "" {
			sb.WriteString(metadataHeader)
			headerWritten = true
		} else if i > 0 {
			sb.WriteString("\n---\n")
		}
		
		sb.WriteString(content)
		
		source := res.Metadata["source_file"]
		if source != "" {
			sources = append(sources, source)
		}
	}

	contextText := sb.String()

	s.logger.Info().
		Int("results_count", len(results)).
		Str("context_content", contextText). // <-- LOG CONTENT RAG PADA FILTERED
		Float32("top_score", results[0].Score).
		Dur("ms", time.Since(start)).
		Msg("[retrieval] filtered search done")

	return PromptContext{
		Context:    contextText,
		Sources:    sources,
		HasResults: true,
	}, nil
}
