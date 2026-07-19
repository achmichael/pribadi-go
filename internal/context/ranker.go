package context

import (
	"sort"
	"strings"

	"github.com/rs/zerolog"
)

// ─── Types ─────────────────────────────────────────────────────────

// RankedChunk represents a RAG chunk with a relevance score.
type RankedChunk struct {
	Content  string
	Score    float32
	Source   string
	Metadata map[string]string
}

// ─── Interface ─────────────────────────────────────────────────────

// Ranker scores and ranks context chunks by relevance.
type Ranker interface {
	// Rank sorts chunks by relevance and filters low-quality ones.
	Rank(chunks []RankedChunk, query string, maxChunks int) []RankedChunk
}

// ─── Implementation ────────────────────────────────────────────────

type ranker struct {
	logger *zerolog.Logger
}

// NewRanker creates a context Ranker.
func NewRanker(logger *zerolog.Logger) Ranker {
	return &ranker{logger: logger}
}

// minRelevanceScore is the minimum score to include a chunk.
const minRelevanceScore float32 = 0.25

func (r *ranker) Rank(chunks []RankedChunk, query string, maxChunks int) []RankedChunk {
	if len(chunks) == 0 {
		return nil
	}

	// 1. Apply keyword boost
	queryTerms := extractTerms(query)
	for i := range chunks {
		boost := r.keywordBoost(chunks[i].Content, queryTerms)
		chunks[i].Score += boost
	}

	// 2. Sort by score descending
	sort.Slice(chunks, func(i, j int) bool {
		return chunks[i].Score > chunks[j].Score
	})

	// 3. Filter below threshold
	var filtered []RankedChunk
	for _, c := range chunks {
		if c.Score < minRelevanceScore {
			continue
		}
		filtered = append(filtered, c)
		if len(filtered) >= maxChunks {
			break
		}
	}

	r.logger.Debug().
		Int("input_chunks", len(chunks)).
		Int("output_chunks", len(filtered)).
		Msg("[ranker] ranked")

	return filtered
}

// keywordBoost adds a small score boost for chunks containing query terms.
func (r *ranker) keywordBoost(content string, queryTerms []string) float32 {
	if len(queryTerms) == 0 {
		return 0
	}

	lower := strings.ToLower(content)
	matches := 0
	for _, term := range queryTerms {
		if strings.Contains(lower, term) {
			matches++
		}
	}

	// Boost: up to 0.1 based on keyword overlap
	ratio := float32(matches) / float32(len(queryTerms))
	return ratio * 0.1
}

// extractTerms splits a query into meaningful terms (>2 chars, lowered).
func extractTerms(query string) []string {
	words := strings.Fields(strings.ToLower(query))
	// Common stopwords to skip
	stops := map[string]bool{
		"apa": true, "yang": true, "dan": true, "di": true, "dari": true,
		"ini": true, "itu": true, "untuk": true, "dengan": true, "ke": true,
		"pada": true, "adalah": true, "the": true, "a": true, "an": true,
		"is": true, "are": true, "of": true, "in": true, "to": true,
		"and": true, "for": true, "with": true, "on": true, "at": true,
		"by": true, "this": true, "that": true, "it": true, "was": true,
	}

	var terms []string
	for _, w := range words {
		w = strings.Trim(w, ".,!?;:\"'()")
		if len(w) <= 2 || stops[w] {
			continue
		}
		terms = append(terms, w)
	}
	return terms
}
