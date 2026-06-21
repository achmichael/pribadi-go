package rag

import (
	"context"
	"strings"

	"github.com/achmichael/pribadi-go/internal/repository"
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
}

// NewRetrievalService creates a new retrieval service
func NewRetrievalService(vectorRepo repository.VectorRepository) RetrievalService {
	return &retrievalService{
		vectorRepo: vectorRepo,
	}
}

// Retrieve searches top relevant chunks
func (s *retrievalService) Retrieve(ctx context.Context, question string) (PromptContext, error) {
	results, err := s.vectorRepo.Search(ctx, question, 5)
	if err != nil {
		return PromptContext{}, err
	}

	if len(results) == 0 {
		return PromptContext{HasResults: false}, nil
	}

	var sb strings.Builder
	var sources []string

	for i, res := range results {
		if i > 0 {
			sb.WriteString("\n---\n")
		}
		sb.WriteString(res.Content)

		source := res.Metadata["source_file"]
		if source != "" {
			sources = append(sources, source)
		}
	}

	return PromptContext{
		Context:    sb.String(),
		Sources:    sources,
		HasResults: true,
	}, nil
}
