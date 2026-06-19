package repository

import (
	"context"
	"fmt"

	"github.com/achmichael/pribadi-go/pkg/utils"
	"github.com/philippgille/chromem-go"
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
}

// NewVectorRepository creates a new vector repository using chromem-go
func NewVectorRepository(persistPath string, ollamaBaseURL string) (VectorRepository, error) {
	// Create chromem DB with persistence
	db, err := chromem.NewPersistentDB(persistPath, false)
	if err != nil {
		return nil, fmt.Errorf("failed to create chromem DB: %w", err)
	}

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
		// Collection doesn't exist, create it
		_, err = db.CreateCollection(collName, nil, embeddingFunc)
		if err != nil {
			return nil, fmt.Errorf("failed to create collection: %w", err)
		}
	}

	return &chromemVectorRepo{
		db:       db,
		embedder: embedder,
		collName: collName,
	}, nil
}

func (r *chromemVectorRepo) UpsertDocument(ctx context.Context, id string, content string, metadata map[string]string) error {
	coll := r.db.GetCollection(r.collName, nil)
	if coll == nil {
		return fmt.Errorf("collection not found")
	}

	// Generate embedding
	embedding, err := r.embedder.Embed(ctx, content)
	if err != nil {
		return fmt.Errorf("failed to generate embedding: %w", err)
	}

	// Add document to collection
	err = coll.AddDocument(ctx, chromem.Document{
		ID:        id,
		Content:   content,
		Metadata:  metadata,
		Embedding: embedding,
	})
	if err != nil {
		return fmt.Errorf("failed to add document: %w", err)
	}

	return nil
}

func (r *chromemVectorRepo) Search(ctx context.Context, query string, topK int) ([]SearchResult, error) {
	coll := r.db.GetCollection(r.collName, nil)
	if coll == nil {
		return nil, fmt.Errorf("collection not found")
	}

	// Generate query embedding
	queryEmbedding, err := r.embedder.Embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// Search in collection
	results, err := coll.QueryEmbedding(ctx, queryEmbedding, topK, nil, nil)
	if err != nil {
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

	return searchResults, nil
}

func (r *chromemVectorRepo) Close() error {
	// chromem-go handles persistence automatically, no explicit close needed
	return nil
}
