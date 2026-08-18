// Package prompt provides citation and tool result formatting.
package prompt

import (
	"fmt"
	"strings"
)

// ─── Citation Formatter ────────────────────────────────────────────

// CitationStyle defines how citations are formatted.
type CitationStyle string

const (
	CitationInline   CitationStyle = "inline"   // [1], [2]
	CitationFootnote CitationStyle = "footnote" // numbered references at end
	CitationNone     CitationStyle = "none"     // no citations
)

// Citation represents a source reference.
type Citation struct {
	Number int
	Source string
	Type   string // "document", "memory", "web"
}

// CitationFormatter formats citations in responses.
type CitationFormatter interface {
	// Format applies citations to response text.
	Format(response string, citations []Citation, style CitationStyle) string
}

type citationFormatter struct{}

// NewCitationFormatter creates a CitationFormatter.
func NewCitationFormatter() CitationFormatter {
	return &citationFormatter{}
}

func (f *citationFormatter) Format(response string, citations []Citation, style CitationStyle) string {
	if style == CitationNone || len(citations) == 0 {
		return response
	}

	switch style {
	case CitationFootnote:
		return f.formatFootnote(response, citations)
	case CitationInline:
		return f.formatInline(response, citations)
	default:
		return response
	}
}

func (f *citationFormatter) formatInline(response string, citations []Citation) string {
	// Citations already embedded in response text as [1], [2], etc.
	// Just append the reference list
	var sb strings.Builder
	sb.WriteString(response)
	sb.WriteString("\n\n---\nSources:\n")
	for _, c := range citations {
		displaySrc := c.Source
		if len(displaySrc) > 20 && !strings.Contains(displaySrc, ".") && !strings.Contains(displaySrc, " ") {
			displaySrc = "Dokumen Referensi"
		}
		sb.WriteString(fmt.Sprintf("[%d] %s (%s)\n", c.Number, displaySrc, c.Type))
	}
	return sb.String()
}

func (f *citationFormatter) formatFootnote(response string, citations []Citation) string {
	var sb strings.Builder
	sb.WriteString(response)
	sb.WriteString("\n\n---\nReferences:\n")
	for _, c := range citations {
		displaySrc := c.Source
		if len(displaySrc) > 20 && !strings.Contains(displaySrc, ".") && !strings.Contains(displaySrc, " ") {
			displaySrc = "Dokumen Referensi"
		}
		sb.WriteString(fmt.Sprintf("%d. %s [%s]\n", c.Number, displaySrc, c.Type))
	}
	return sb.String()
}

// ─── Tool Result Injector ──────────────────────────────────────────

// ToolResult represents output from an external tool.
type ToolResult struct {
	ToolName string
	Output   string
	Success  bool
	Error    string
}

// ToolInjector formats tool results for inclusion in prompts.
type ToolInjector interface {
	// InjectBefore adds tool results before user message.
	InjectBefore(userMessage string, results []ToolResult) string

	// InjectAfter adds tool results after system prompt.
	InjectAfter(systemPrompt string, results []ToolResult) string
}

type toolInjector struct{}

// NewToolInjector creates a ToolInjector.
func NewToolInjector() ToolInjector {
	return &toolInjector{}
}

func (t *toolInjector) InjectBefore(userMessage string, results []ToolResult) string {
	if len(results) == 0 {
		return userMessage
	}

	var sb strings.Builder
	sb.WriteString("=== Tool Execution Results ===\n")
	for _, r := range results {
		sb.WriteString(fmt.Sprintf("\n[Tool: %s]\n", r.ToolName))
		if r.Success {
			sb.WriteString(fmt.Sprintf("Status: Success\nOutput:\n%s\n", r.Output))
		} else {
			sb.WriteString(fmt.Sprintf("Status: Failed\nError: %s\n", r.Error))
		}
	}
	sb.WriteString("\n=== End Tool Results ===\n\n")
	sb.WriteString(userMessage)
	return sb.String()
}

func (t *toolInjector) InjectAfter(systemPrompt string, results []ToolResult) string {
	if len(results) == 0 {
		return systemPrompt
	}

	var sb strings.Builder
	sb.WriteString(systemPrompt)
	sb.WriteString("\n\n=== Available Tool Results ===\n")
	for _, r := range results {
		sb.WriteString(fmt.Sprintf("\n[%s]: ", r.ToolName))
		if r.Success {
			sb.WriteString(r.Output)
		} else {
			sb.WriteString(fmt.Sprintf("(failed: %s)", r.Error))
		}
		sb.WriteString("\n")
	}
	sb.WriteString("=== End Tool Results ===\n")
	return sb.String()
}

// ─── Memory Injection ──────────────────────────────────────────────

// MemoryItem represents a fact with confidence.
type MemoryItem struct {
	Fact       string
	Confidence string // "high", "medium", "low"
	Category   string
}

// MemoryInjector formats memory facts for prompt injection.
type MemoryInjector interface {
	// FormatForPrompt converts memory items to prompt-ready text.
	FormatForPrompt(items []MemoryItem, includeConfidence bool) string
}

type memoryInjector struct{}

// NewMemoryInjector creates a MemoryInjector.
func NewMemoryInjector() MemoryInjector {
	return &memoryInjector{}
}

func (m *memoryInjector) FormatForPrompt(items []MemoryItem, includeConfidence bool) string {
	if len(items) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("<memory-context>\n")
	sb.WriteString("Fakta relevan tentang pengguna:\n")

	// Group by category
	categories := make(map[string][]MemoryItem)
	for _, item := range items {
		cat := item.Category
		if cat == "" {
			cat = "general"
		}
		categories[cat] = append(categories[cat], item)
	}

	for cat, facts := range categories {
		if len(categories) > 1 {
			sb.WriteString(fmt.Sprintf("\n[%s]\n", cat))
		}
		for _, f := range facts {
			if includeConfidence {
				sb.WriteString(fmt.Sprintf("- %s (confidence: %s)\n", f.Fact, f.Confidence))
			} else {
				sb.WriteString(fmt.Sprintf("- %s\n", f.Fact))
			}
		}
	}
	sb.WriteString("</memory-context>")
	return sb.String()
}
