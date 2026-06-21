package chunker

import (
	"strings"
	"testing"
)

func TestChunkText_EmptyInput(t *testing.T) {
	chunks, err := ChunkText("", "test.docx", ChunkOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chunks) != 0 {
		t.Fatalf("expected 0 chunks, got %d", len(chunks))
	}
}

func TestChunkText_ShortText(t *testing.T) {
	text := "Hello world. This is a short document."
	chunks, err := ChunkText(text, "short.docx", ChunkOptions{MaxTokens: 400, Overlap: 75})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	if chunks[0].Content != text {
		t.Errorf("content mismatch: %q", chunks[0].Content)
	}
	if chunks[0].TotalChunks != 1 {
		t.Errorf("TotalChunks should be 1, got %d", chunks[0].TotalChunks)
	}
	if chunks[0].SourceFile != "short.docx" {
		t.Errorf("SourceFile mismatch: %q", chunks[0].SourceFile)
	}
}

func TestChunkText_NoChunkExceedsMaxTokens(t *testing.T) {
	// Build a long text: ~80 sentences, each ~20 tokens
	var sb strings.Builder
	sentence := "Ini adalah kalimat percobaan yang cukup panjang untuk pengujian chunking dokumen. "
	for i := 0; i < 80; i++ {
		sb.WriteString(sentence)
		if i%5 == 4 {
			sb.WriteString("\n\n")
		}
	}

	opts := ChunkOptions{MaxTokens: 400, Overlap: 75}
	chunks, err := ChunkText(sb.String(), "long.pdf", opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(chunks) < 2 {
		t.Fatalf("expected multiple chunks, got %d", len(chunks))
	}

	for i, c := range chunks {
		tokens := estimateTokens(c.Content)
		// Allow some tolerance for overlap prefix
		maxAllowed := opts.MaxTokens + opts.Overlap + 10
		if tokens > maxAllowed {
			t.Errorf("chunk %d exceeds max: %d tokens (max allowed %d)\ncontent length: %d chars",
				i, tokens, maxAllowed, len(c.Content))
		}
		if c.ChunkIndex != i {
			t.Errorf("chunk %d has wrong ChunkIndex %d", i, c.ChunkIndex)
		}
		if c.TotalChunks != len(chunks) {
			t.Errorf("chunk %d has wrong TotalChunks %d (expected %d)", i, c.TotalChunks, len(chunks))
		}
		if c.SourceFile != "long.pdf" {
			t.Errorf("chunk %d has wrong SourceFile %q", i, c.SourceFile)
		}
	}
}

func TestChunkText_OverlapPresent(t *testing.T) {
	// Build text with clear paragraph breaks
	paragraphs := []string{
		"Paragraf pertama berisi informasi tentang proyek ini.",
		"Paragraf kedua membahas arsitektur sistem yang digunakan.",
		"Paragraf ketiga menjelaskan detail implementasi.",
		"Paragraf keempat berisi kesimpulan dari dokumen.",
	}
	text := strings.Join(paragraphs, "\n\n")

	opts := ChunkOptions{MaxTokens: 50, Overlap: 20}
	chunks, err := ChunkText(text, "overlap.docx", opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(chunks) < 2 {
		t.Skipf("text too short to produce multiple chunks with these opts, got %d", len(chunks))
	}

	// Verify overlap: chunk[i] should share some text with chunk[i-1]
	foundOverlap := false
	for i := 1; i < len(chunks); i++ {
		prev := chunks[i-1].Content
		curr := chunks[i].Content
		// Check if any sentence from prev appears in curr
		prevSentences := splitSentences(prev)
		for _, s := range prevSentences {
			if strings.Contains(curr, s) {
				foundOverlap = true
				break
			}
		}
	}
	if !foundOverlap {
		t.Error("expected overlap between consecutive chunks but found none")
	}
}

func TestChunkText_NeverSplitsMidSentence(t *testing.T) {
	// One giant paragraph, no newlines, multiple sentences
	text := "Kalimat pertama tentang AI. Kalimat kedua tentang machine learning. " +
		"Kalimat ketiga tentang deep learning. Kalimat keempat tentang neural network. " +
		"Kalimat kelima tentang transformers. Kalimat keenam tentang attention mechanism. " +
		"Kalimat ketujuh tentang embedding. Kalimat kedelapan tentang vector database."

	opts := ChunkOptions{MaxTokens: 60, Overlap: 0}
	chunks, err := ChunkText(text, "test.pdf", opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for i, c := range chunks {
		content := strings.TrimSpace(c.Content)
		// Each chunk should end with a sentence-ending punctuation or be the last chunk
		if i < len(chunks)-1 {
			lastChar := content[len(content)-1]
			if lastChar != '.' && lastChar != '!' && lastChar != '?' {
				t.Errorf("chunk %d does not end at sentence boundary: ...%q",
					i, content[max(0, len(content)-30):])
			}
		}
	}
}

func TestChunkText_MetadataCorrect(t *testing.T) {
	text := strings.Repeat("Beberapa teks panjang untuk pengujian metadata. ", 100)
	chunks, err := ChunkText(text, "meta.pdf", ChunkOptions{MaxTokens: 200, Overlap: 50})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for i, c := range chunks {
		if c.ChunkIndex != i {
			t.Errorf("ChunkIndex: expected %d, got %d", i, c.ChunkIndex)
		}
		if c.TotalChunks != len(chunks) {
			t.Errorf("TotalChunks: expected %d, got %d", len(chunks), c.TotalChunks)
		}
		if c.SourceFile != "meta.pdf" {
			t.Errorf("SourceFile: expected meta.pdf, got %s", c.SourceFile)
		}
	}
}

func TestEstimateTokens(t *testing.T) {
	// "hello" = 5 chars → ceil(5/4) = 2
	if got := estimateTokens("hello"); got != 2 {
		t.Errorf("estimateTokens(hello) = %d, want 2", got)
	}
	// Empty
	if got := estimateTokens(""); got != 0 {
		t.Errorf("estimateTokens('') = %d, want 0", got)
	}
	// Indonesian text with multi-byte chars
	indo := "Bahasa Indonesia"
	expected := (len([]rune(indo)) + 3) / 4
	if got := estimateTokens(indo); got != expected {
		t.Errorf("estimateTokens(%q) = %d, want %d", indo, got, expected)
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
