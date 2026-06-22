package repository_test

import (
	"context"
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	"github.com/achmichael/pribadi-go/internal/repository"
	"github.com/rs/zerolog"
)

// qdrantAddr returns the Qdrant gRPC address for testing.
// Falls back to localhost:6334.
func qdrantAddr() string {
	if v := os.Getenv("QDRANT_ADDR"); v != "" {
		return v
	}
	return "localhost:6334"
}

// ollamaURL returns the Ollama base URL for testing.
func ollamaURL() string {
	if v := os.Getenv("OLLAMA_BASE_URL"); v != "" {
		return v
	}
	return "http://localhost:11434"
}

// isQdrantReachable checks if Qdrant gRPC port is open.
func isQdrantReachable(addr string) bool {
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func TestQdrantVectorRepo_Integration(t *testing.T) {
	addr := qdrantAddr()
	if !isQdrantReachable(addr) {
		t.Skipf("Qdrant not reachable at %s — skipping integration test", addr)
	}

	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}).
		With().Timestamp().Logger()

	repo, err := repository.NewVectorRepository(addr, ollamaURL(), &logger)
	if err != nil {
		t.Fatalf("NewVectorRepository failed: %v", err)
	}
	defer repo.Close()

	ctx := context.Background()

	// ── Test 1: Upsert ──────────────────────────────────────────────
	t.Run("Upsert", func(t *testing.T) {
		err := repo.UpsertDocument(ctx, "test-doc-001", "Qdrant is a high-performance vector database written in Rust.", map[string]string{
			"source_file": "test.txt",
			"chunk_index": "0",
		})
		if err != nil {
			t.Fatalf("UpsertDocument failed: %v", err)
		}
	})

	// ── Test 2: Upsert multiple docs ────────────────────────────────
	t.Run("UpsertMultiple", func(t *testing.T) {
		docs := []struct {
			id      string
			content string
		}{
			{"test-doc-002", "Golang is a statically typed programming language designed at Google."},
			{"test-doc-003", "Ollama allows running large language models locally on your machine."},
			{"test-doc-004", "WhatsApp is a messaging platform used by billions of people worldwide."},
		}

		for _, d := range docs {
			err := repo.UpsertDocument(ctx, d.id, d.content, map[string]string{
				"source_file": "test.txt",
			})
			if err != nil {
				t.Fatalf("UpsertDocument(%s) failed: %v", d.id, err)
			}
		}
	})

	// Small delay for Qdrant indexing
	time.Sleep(500 * time.Millisecond)

	// ── Test 3: Search returns results ──────────────────────────────
	t.Run("SearchReturnsResults", func(t *testing.T) {
		results, err := repo.Search(ctx, "vector database", 3)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		if len(results) == 0 {
			t.Fatal("Search returned 0 results, expected > 0")
		}

		t.Logf("Search returned %d results", len(results))
		for i, r := range results {
			t.Logf("  [%d] id=%s score=%.4f content=%q", i, r.ID, r.Score, truncate(r.Content, 80))
		}

		// Top result should be about Qdrant/vector database
		if results[0].Score < 0.3 {
			t.Errorf("Top result score %.4f too low, expected >= 0.3", results[0].Score)
		}
	})

	// ── Test 4: Search result contains metadata ─────────────────────
	t.Run("SearchHasMetadata", func(t *testing.T) {
		results, err := repo.Search(ctx, "vector database Rust", 1)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		if len(results) == 0 {
			t.Fatal("no results")
		}

		r := results[0]
		if r.Content == "" {
			t.Error("result Content is empty")
		}
		if r.Metadata["source_file"] != "test.txt" {
			t.Errorf("expected source_file=test.txt, got %q", r.Metadata["source_file"])
		}
		t.Logf("Result: id=%s score=%.4f metadata=%v", r.ID, r.Score, r.Metadata)
	})

	// ── Test 5: Upsert same ID updates (not duplicate) ──────────────
	t.Run("UpsertIdempotent", func(t *testing.T) {
		err := repo.UpsertDocument(ctx, "test-doc-001", "Updated: Qdrant is a vector search engine.", map[string]string{
			"source_file": "updated.txt",
		})
		if err != nil {
			t.Fatalf("UpsertDocument (update) failed: %v", err)
		}

		time.Sleep(300 * time.Millisecond)

		results, err := repo.Search(ctx, "Qdrant vector search engine", 1)
		if err != nil {
			t.Fatalf("Search after update failed: %v", err)
		}

		if len(results) == 0 {
			t.Fatal("no results after update")
		}

		if results[0].Content != "Updated: Qdrant is a vector search engine." {
			t.Errorf("expected updated content, got %q", results[0].Content)
		}
		if results[0].Metadata["source_file"] != "updated.txt" {
			t.Errorf("expected updated metadata, got %q", results[0].Metadata["source_file"])
		}
	})

	// ── Test 6: Search with no matching results ─────────────────────
	t.Run("SearchIrrelevant", func(t *testing.T) {
		results, err := repo.Search(ctx, "quantum physics black holes entropy", 3)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		// Should return results but with low scores
		t.Logf("Irrelevant search returned %d results", len(results))
		for i, r := range results {
			t.Logf("  [%d] score=%.4f content=%q", i, r.Score, truncate(r.Content, 60))
		}
	})

	// ── Cleanup: delete test collection points ──────────────────────
	t.Run("Cleanup", func(t *testing.T) {
		// We don't delete the collection since other data might exist.
		// In a real CI you would use a test-specific collection name.
		t.Log("Test documents left in collection (manual cleanup if needed)")
	})
}

func TestQdrantVectorRepo_ConnectionFailure(t *testing.T) {
	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout}).With().Timestamp().Logger()

	// Connect to a port where nothing listens
	_, err := repository.NewVectorRepository("localhost:19999", ollamaURL(), &logger)
	if err == nil {
		t.Fatal("expected error connecting to bad address, got nil")
	}
	t.Logf("Got expected error: %v", err)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + fmt.Sprintf("...(%d more)", len(s)-n)
}
