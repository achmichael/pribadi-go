package context

import (
	"os"
	"testing"

	"github.com/rs/zerolog"
)

func TestRanker_SortByScore(t *testing.T) {
	logger := zerolog.New(os.Stderr).With().Logger()
	r := NewRanker(&logger)

	chunks := []RankedChunk{
		{Content: "low relevance", Score: 0.3},
		{Content: "high relevance", Score: 0.9},
		{Content: "medium relevance", Score: 0.6},
	}

	result := r.Rank(chunks, "test query", 10)

	if len(result) != 3 {
		t.Fatalf("expected 3 chunks, got %d", len(result))
	}
	if result[0].Content != "high relevance" {
		t.Errorf("expected highest score first, got %s", result[0].Content)
	}
}

func TestRanker_FilterBelowThreshold(t *testing.T) {
	logger := zerolog.New(os.Stderr).With().Logger()
	r := NewRanker(&logger)

	chunks := []RankedChunk{
		{Content: "good", Score: 0.5},
		{Content: "bad", Score: 0.1},
		{Content: "terrible", Score: 0.05},
	}

	result := r.Rank(chunks, "query", 10)

	if len(result) != 1 {
		t.Errorf("expected 1 chunk above threshold, got %d", len(result))
	}
}

func TestRanker_MaxChunks(t *testing.T) {
	logger := zerolog.New(os.Stderr).With().Logger()
	r := NewRanker(&logger)

	chunks := []RankedChunk{
		{Content: "a", Score: 0.9},
		{Content: "b", Score: 0.8},
		{Content: "c", Score: 0.7},
		{Content: "d", Score: 0.6},
	}

	result := r.Rank(chunks, "query", 2)

	if len(result) != 2 {
		t.Errorf("expected max 2 chunks, got %d", len(result))
	}
}

func TestRanker_KeywordBoost(t *testing.T) {
	logger := zerolog.New(os.Stderr).With().Logger()
	r := NewRanker(&logger)

	chunks := []RankedChunk{
		{Content: "This talks about Python programming language", Score: 0.5},
		{Content: "This is about cooking recipes", Score: 0.55},
	}

	result := r.Rank(chunks, "Python programming tutorial", 10)

	// Python chunk should be boosted above cooking
	if result[0].Content != "This talks about Python programming language" {
		t.Error("expected keyword-boosted chunk first")
	}
}

func TestRanker_Empty(t *testing.T) {
	logger := zerolog.New(os.Stderr).With().Logger()
	r := NewRanker(&logger)

	result := r.Rank(nil, "query", 10)
	if result != nil {
		t.Error("expected nil for empty input")
	}
}

func TestExtractTerms(t *testing.T) {
	terms := extractTerms("Apa itu machine learning untuk pemula?")

	// Should filter stopwords "apa", "itu", "untuk"
	found := map[string]bool{}
	for _, term := range terms {
		found[term] = true
	}

	if found["apa"] {
		t.Error("stopword 'apa' should be filtered")
	}
	if !found["machine"] {
		t.Error("expected 'machine' in terms")
	}
	if !found["learning"] {
		t.Error("expected 'learning' in terms")
	}
	if !found["pemula"] {
		t.Error("expected 'pemula' in terms")
	}
}
