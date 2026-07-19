package prompt

import (
	"testing"
)

func TestCitationFormatter_Inline(t *testing.T) {
	formatter := NewCitationFormatter()

	response := "Machine learning is a subset of AI [1]. Deep learning uses neural networks [2]."
	citations := []Citation{
		{Number: 1, Source: "doc1.pdf", Type: "document"},
		{Number: 2, Source: "doc2.pdf", Type: "document"},
	}

	result := formatter.Format(response, citations, CitationInline)

	if result == response {
		t.Error("expected citations appended")
	}
	if !contains(result, "Sources:") {
		t.Error("expected Sources section")
	}
	if !contains(result, "[1] doc1.pdf") {
		t.Error("expected citation [1]")
	}
}

func TestCitationFormatter_Footnote(t *testing.T) {
	formatter := NewCitationFormatter()

	response := "Some text about AI."
	citations := []Citation{
		{Number: 1, Source: "book.pdf", Type: "document"},
	}

	result := formatter.Format(response, citations, CitationFootnote)

	if !contains(result, "References:") {
		t.Error("expected References section")
	}
	if !contains(result, "1. book.pdf") {
		t.Error("expected footnote style citation")
	}
}

func TestCitationFormatter_None(t *testing.T) {
	formatter := NewCitationFormatter()

	response := "Original text"
	citations := []Citation{
		{Number: 1, Source: "doc.pdf", Type: "document"},
	}

	result := formatter.Format(response, citations, CitationNone)

	if result != response {
		t.Error("expected no changes with CitationNone")
	}
}

func TestCitationFormatter_EmptyCitations(t *testing.T) {
	formatter := NewCitationFormatter()

	response := "Text without citations"
	result := formatter.Format(response, nil, CitationInline)

	if result != response {
		t.Error("expected no changes with empty citations")
	}
}

func TestToolInjector_InjectBefore(t *testing.T) {
	injector := NewToolInjector()

	userMsg := "Show me the results"
	results := []ToolResult{
		{ToolName: "calculator", Output: "42", Success: true},
		{ToolName: "search", Output: "", Success: false, Error: "timeout"},
	}

	result := injector.InjectBefore(userMsg, results)

	if !contains(result, "Tool Execution Results") {
		t.Error("expected tool results header")
	}
	if !contains(result, "calculator") {
		t.Error("expected calculator tool")
	}
	if !contains(result, "42") {
		t.Error("expected calculator output")
	}
	if !contains(result, "Failed") {
		t.Error("expected failed status for search")
	}
	if !contains(result, userMsg) {
		t.Error("expected original user message")
	}
}

func TestToolInjector_InjectAfter(t *testing.T) {
	injector := NewToolInjector()

	sysPrompt := "You are an assistant."
	results := []ToolResult{
		{ToolName: "weather", Output: "sunny, 25C", Success: true},
	}

	result := injector.InjectAfter(sysPrompt, results)

	if !contains(result, sysPrompt) {
		t.Error("expected original system prompt")
	}
	if !contains(result, "Available Tool Results") {
		t.Error("expected tool results section")
	}
	if !contains(result, "weather") {
		t.Error("expected weather tool")
	}
}

func TestToolInjector_Empty(t *testing.T) {
	injector := NewToolInjector()

	msg := "original"
	result := injector.InjectBefore(msg, nil)
	if result != msg {
		t.Error("expected no changes with empty results")
	}

	result = injector.InjectAfter(msg, []ToolResult{})
	if result != msg {
		t.Error("expected no changes with empty results")
	}
}

func TestMemoryInjector_Basic(t *testing.T) {
	injector := NewMemoryInjector()

	items := []MemoryItem{
		{Fact: "User likes Python", Confidence: "high", Category: "preferences"},
		{Fact: "User is a developer", Confidence: "high", Category: "profile"},
	}

	result := injector.FormatForPrompt(items, true)

	if !contains(result, "<memory-context>") {
		t.Error("expected memory context tags")
	}
	if !contains(result, "User likes Python") {
		t.Error("expected first fact")
	}
	if !contains(result, "confidence: high") {
		t.Error("expected confidence when includeConfidence=true")
	}
	if !contains(result, "[preferences]") {
		t.Error("expected category grouping")
	}
}

func TestMemoryInjector_WithoutConfidence(t *testing.T) {
	injector := NewMemoryInjector()

	items := []MemoryItem{
		{Fact: "User is 25 years old", Confidence: "medium", Category: "profile"},
	}

	result := injector.FormatForPrompt(items, false)

	if contains(result, "confidence:") {
		t.Error("expected no confidence when includeConfidence=false")
	}
	if !contains(result, "User is 25 years old") {
		t.Error("expected fact text")
	}
}

func TestMemoryInjector_Empty(t *testing.T) {
	injector := NewMemoryInjector()

	result := injector.FormatForPrompt(nil, true)
	if result != "" {
		t.Error("expected empty string for nil items")
	}

	result = injector.FormatForPrompt([]MemoryItem{}, false)
	if result != "" {
		t.Error("expected empty string for empty items")
	}
}

func TestMemoryInjector_NoCategory(t *testing.T) {
	injector := NewMemoryInjector()

	items := []MemoryItem{
		{Fact: "Some fact", Confidence: "low", Category: ""},
	}

	result := injector.FormatForPrompt(items, true)

	if !contains(result, "Some fact") {
		t.Error("expected fact even without category")
	}
	// When there's only one category, it still groups but may not print header
	// The important thing is the fact is present
}

// Helper
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && 
		(s == substr || len(s) >= len(substr) && 
		indexAny(s, substr) >= 0)
}

func indexAny(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
