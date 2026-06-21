package chunker

import (
	"strings"
	"unicode"
)

// Chunk represents a single chunk of a larger document.
type Chunk struct {
	Content     string
	SourceFile  string
	ChunkIndex  int
	TotalChunks int
}

// ChunkOptions controls the chunking behaviour.
type ChunkOptions struct {
	MaxTokens int // target max tokens per chunk  (default 400)
	Overlap   int // overlap tokens between chunks (default 75)
}

func (o ChunkOptions) withDefaults() ChunkOptions {
	if o.MaxTokens <= 0 {
		o.MaxTokens = 400
	}
	if o.Overlap <= 0 {
		o.Overlap = 75
	}
	if o.Overlap >= o.MaxTokens {
		o.Overlap = o.MaxTokens / 4
	}
	return o
}

// estimateTokens approximates token count: chars/4.
// Good enough for Latin + Indonesian text with nomic-embed-text.
func estimateTokens(s string) int {
	n := 0
	for range s {
		n++
	}
	return (n + 3) / 4 // ceil(len_runes / 4)
}

// ChunkText splits text into overlapping chunks that respect paragraph and
// sentence boundaries. It never splits mid-sentence.
//
// Strategy (recursive):
//  1. Split on paragraph boundaries (\n\n).
//  2. If a single paragraph exceeds MaxTokens, split on sentence boundaries
//     (period/exclamation/question followed by space or newline).
//  3. Merge small consecutive segments until MaxTokens is reached.
//  4. Apply overlap by re-including trailing sentences from previous chunk.
func ChunkText(text string, sourceFile string, opts ChunkOptions) ([]Chunk, error) {
	opts = opts.withDefaults()

	// Normalise whitespace edges
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, nil
	}

	// Step 1: split into paragraphs
	paragraphs := splitParagraphs(text)

	// Step 2: break large paragraphs into sentences
	var segments []string
	for _, p := range paragraphs {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if estimateTokens(p) <= opts.MaxTokens {
			segments = append(segments, p)
		} else {
			sentences := splitSentences(p)
			segments = append(segments, sentences...)
		}
	}

	if len(segments) == 0 {
		return nil, nil
	}

	// Step 3: greedy-merge segments into chunks respecting MaxTokens
	raw := mergeSegments(segments, opts.MaxTokens)

	// Step 4: add overlap by prepending tail of previous chunk
	chunks := applyOverlap(raw, segments, opts)

	// Build result with metadata
	result := make([]Chunk, len(chunks))
	for i, c := range chunks {
		result[i] = Chunk{
			Content:     strings.TrimSpace(c),
			SourceFile:  sourceFile,
			ChunkIndex:  i,
			TotalChunks: len(chunks),
		}
	}
	return result, nil
}

// splitParagraphs splits on double-newline (blank line).
func splitParagraphs(text string) []string {
	// Normalise \r\n → \n
	text = strings.ReplaceAll(text, "\r\n", "\n")
	return strings.Split(text, "\n\n")
}

// splitSentences splits a paragraph into sentences.
// Handles ". ", "! ", "? " and also ".\n", etc.
func splitSentences(paragraph string) []string {
	var sentences []string
	var buf strings.Builder
	runes := []rune(paragraph)

	for i := 0; i < len(runes); i++ {
		buf.WriteRune(runes[i])
		if isSentenceEnd(runes, i) {
			s := strings.TrimSpace(buf.String())
			if s != "" {
				sentences = append(sentences, s)
			}
			buf.Reset()
		}
	}
	// Remaining text
	if s := strings.TrimSpace(buf.String()); s != "" {
		sentences = append(sentences, s)
	}
	if len(sentences) == 0 {
		sentences = []string{paragraph}
	}
	return sentences
}

// isSentenceEnd returns true when runes[i] is a sentence-ending punctuation
// followed by whitespace or end-of-text.
func isSentenceEnd(runes []rune, i int) bool {
	r := runes[i]
	if r != '.' && r != '!' && r != '?' {
		return false
	}
	// End of text counts
	if i+1 >= len(runes) {
		return true
	}
	next := runes[i+1]
	return unicode.IsSpace(next)
}

// mergeSegments greedily packs segments into chunks without exceeding max tokens.
func mergeSegments(segments []string, maxTokens int) []string {
	var chunks []string
	var buf strings.Builder
	bufTokens := 0

	for _, seg := range segments {
		segTokens := estimateTokens(seg)

		// If single segment already exceeds max, it becomes its own chunk
		if segTokens > maxTokens {
			// Flush buffer first
			if bufTokens > 0 {
				chunks = append(chunks, buf.String())
				buf.Reset()
				bufTokens = 0
			}
			chunks = append(chunks, seg)
			continue
		}

		separator := "\n\n"
		sepTokens := 1 // rough
		if bufTokens == 0 {
			sepTokens = 0
			separator = ""
		}

		if bufTokens+sepTokens+segTokens > maxTokens {
			// Flush
			chunks = append(chunks, buf.String())
			buf.Reset()
			bufTokens = 0
			separator = ""
			sepTokens = 0
		}

		buf.WriteString(separator)
		buf.WriteString(seg)
		bufTokens += sepTokens + segTokens
	}
	if bufTokens > 0 {
		chunks = append(chunks, buf.String())
	}
	return chunks
}

// applyOverlap re-adds trailing sentences from the previous chunk as a prefix
// to achieve the desired overlap.
func applyOverlap(raw []string, _ []string, opts ChunkOptions) []string {
	if len(raw) <= 1 || opts.Overlap <= 0 {
		return raw
	}

	result := make([]string, len(raw))
	result[0] = raw[0]

	for i := 1; i < len(raw); i++ {
		// Take trailing sentences from previous raw chunk as overlap prefix
		prevSentences := splitSentences(raw[i-1])

		var overlap strings.Builder
		overlapTokens := 0
		// Walk backwards through previous chunk's sentences
		for j := len(prevSentences) - 1; j >= 0; j-- {
			st := estimateTokens(prevSentences[j])
			if overlapTokens+st > opts.Overlap {
				break
			}
			overlapTokens += st
			if overlap.Len() > 0 {
				// Prepend: we build reversed then flip
				overlap.WriteString(" ")
			}
			overlap.WriteString(prevSentences[j])
		}

		if overlap.Len() > 0 {
			// The overlap was built in reverse sentence order; rebuild correctly
			overlapSentences := splitSentences(overlap.String())
			// Reverse
			for l, r := 0, len(overlapSentences)-1; l < r; l, r = l+1, r-1 {
				overlapSentences[l], overlapSentences[r] = overlapSentences[r], overlapSentences[l]
			}
			prefix := strings.Join(overlapSentences, " ")
			result[i] = prefix + "\n\n" + raw[i]
		} else {
			result[i] = raw[i]
		}
	}
	return result
}
