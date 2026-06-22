package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/achmichael/pribadi-go/pkg/utils"
	"github.com/philippgille/chromem-go"
	"github.com/rs/zerolog"
)

// SearchResult represents a search result from the vector database
type SearchResult struct {
	ID       string
	Content  string
	Score    float32
	Metadata map[string]string
}

// VectorRepository defines the interface for vector database operations
type VectorRepository interface {
	UpsertDocument(ctx context.Context, id string, content string, metadata map[string]string) error
	Search(ctx context.Context, query string, topK int) ([]SearchResult, error)
	Close() error
}

type chromemVectorRepo struct {
	db       *chromem.DB
	embedder *utils.OllamaEmbedder
	collName string
	logger   *zerolog.Logger
}

// NewVectorRepository creates a new vector repository using chromem-go
func NewVectorRepository(persistPath string, ollamaBaseURL string, logger *zerolog.Logger) (VectorRepository, error) {
	start := time.Now()

	// Create chromem DB with persistence
	db, err := chromem.NewPersistentDB(persistPath, false)
	if err != nil {
		return nil, fmt.Errorf("failed to create chromem DB: %w", err)
	}

	logger.Info().
		Str("path", persistPath).
		Dur("duration_ms", time.Since(start)).
		Msg("[vector_repo] chromem DB opened")

	// Create embedder using nomic-embed-text model
	embedder := utils.NewOllamaEmbedder(ollamaBaseURL, "nomic-embed-text")

	// Create embedding function wrapper
	embeddingFunc := func(ctx context.Context, text string) ([]float32, error) {
		return embedder.Embed(ctx, text)
	}

	// Create or get collection
	collName := "documents"
	coll := db.GetCollection(collName, nil)
	if coll == nil {
		_, err = db.CreateCollection(collName, nil, embeddingFunc)
		if err != nil {
			return nil, fmt.Errorf("failed to create collection: %w", err)
		}
		logger.Info().Str("collection", collName).Msg("[vector_repo] collection created")
	} else {
		logger.Info().Str("collection", collName).Msg("[vector_repo] collection loaded")
	}

	return &chromemVectorRepo{
		db:       db,
		embedder: embedder,
		collName: collName,
		logger:   logger,
	}, nil
}

func (r *chromemVectorRepo) UpsertDocument(ctx context.Context, id string, content string, metadata map[string]string) error {
	start := time.Now()

	coll := r.db.GetCollection(r.collName, nil)
	if coll == nil {
		return fmt.Errorf("collection not found")
	}

	// Generate embedding
	embedStart := time.Now()
	embedding, err := r.embedder.Embed(ctx, content)
	embedDur := time.Since(embedStart)
	if err != nil {
		r.logger.Error().Err(err).
			Str("doc_id", id).
			Int("content_len", len(content)).
			Dur("embed_duration_ms", embedDur).
			Msg("[vector_repo] embedding failed")
		return fmt.Errorf("failed to generate embedding: %w", err)
	}

	r.logger.Debug().
		Str("doc_id", id).
		Int("content_len", len(content)).
		Int("embedding_dim", len(embedding)).
		Dur("embed_duration_ms", embedDur).
		Msg("[vector_repo] embedding generated")

	// Add document to collection
	storeStart := time.Now()
	err = coll.AddDocument(ctx, chromem.Document{
		ID:        id,
		Content:   content,
		Metadata:  metadata,
		Embedding: embedding,
	})
	if err != nil {
		r.logger.Error().Err(err).
			Str("doc_id", id).
			Dur("store_duration_ms", time.Since(storeStart)).
			Msg("[vector_repo] AddDocument failed")
		return fmt.Errorf("failed to add document: %w", err)
	}

	r.logger.Debug().
		Str("doc_id", id).
		Dur("embed_ms", embedDur).
		Dur("store_ms", time.Since(storeStart)).
		Dur("total_ms", time.Since(start)).
		Msg("[vector_repo] upsert done")

	return nil
}

func (r *chromemVectorRepo) Search(ctx context.Context, query string, topK int) ([]SearchResult, error) {
	start := time.Now()

	coll := r.db.GetCollection(r.collName, nil)
	if coll == nil {
		return nil, fmt.Errorf("collection not found")
	}

	// Generate query embedding
	embedStart := time.Now()
	queryEmbedding, err := r.embedder.Embed(ctx, query)
	embedDur := time.Since(embedStart)
	if err != nil {
		r.logger.Error().Err(err).
			Int("query_len", len(query)).
			Dur("embed_duration_ms", embedDur).
			Msg("[vector_repo] query embedding failed")
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	r.logger.Debug().
		Int("query_len", len(query)).
		Dur("embed_ms", embedDur).
		Msg("[vector_repo] query embedding generated")

	// Search in collection
	searchStart := time.Now()
	results, err := coll.QueryEmbedding(ctx, queryEmbedding, topK, nil, nil)
	searchDur := time.Since(searchStart)
	if err != nil {
		r.logger.Error().Err(err).
			Dur("search_duration_ms", searchDur).
			Msg("[vector_repo] QueryEmbedding failed")
		return nil, fmt.Errorf("failed to query collection: %w", err)
	}

	// Convert to SearchResult
	searchResults := make([]SearchResult, len(results))
	for i, res := range results {
		searchResults[i] = SearchResult{
			ID:       res.ID,
			Content:  res.Content,
			Score:    res.Similarity,
			Metadata: res.Metadata,
		}
	}

	r.logger.Info().
		Int("query_len", len(query)).
		Int("top_k", topK).
		Int("results", len(searchResults)).
		Dur("embed_ms", embedDur).
		Dur("search_ms", searchDur).
		Dur("total_ms", time.Since(start)).
		Msg("[vector_repo] search done")

	return searchResults, nil
}

func (r *chromemVectorRepo) Close() error {
	return nil
}
