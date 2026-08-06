package prompt

import (
	"strings"
	"testing"

	"github.com/achmichael/pribadi-go/internal/domain"
)

func TestBuild_DefaultConversational(t *testing.T) {
	b := NewBuilder()

	state := domain.DefaultStateData()
	sc := SessionContext{
		State: &state,
		Persona: Persona{
			Name:        "Pribadi",
			Description: "Asisten AI pribadi",
			Tone:        "casual",
		},
	}

	result := b.Build(sc)

	if !strings.Contains(result, "Pribadi") {
		t.Error("expected persona name in prompt")
	}
	if !strings.Contains(result, "Bahasa Indonesia") {
		t.Error("expected Indonesian language rule")
	}
}

func TestBuild_EnglishLanguage(t *testing.T) {
	b := NewBuilder()

	state := domain.DefaultStateData()
	state.Language = "en"

	sc := SessionContext{
		State:   &state,
		Persona: Persona{Name: "A", Description: "B", Tone: "formal"},
	}

	result := b.Build(sc)

	if !strings.Contains(result, "English") {
		t.Error("expected English language rule")
	}
}

func TestBuild_DocumentIsolation(t *testing.T) {
	b := NewBuilder()

	state := domain.DefaultStateData()
	sc := SessionContext{
		State:         &state,
		Persona:       Persona{Name: "A", Description: "B"},
		TargetDocID:   "doc-123",
		MetadataBlock: "--- METADATA ---\n{\"title\": \"Test\"}\n--- END ---",
		RAGContext:    "Some chunk text",
		HasRAG:        true,
	}

	result := b.Build(sc)

	if !strings.Contains(result, "ATURAN PENGGUNAAN KONTEKS DOKUMEN") {
		t.Error("expected document isolation rules")
	}
	if !strings.Contains(result, "doc-123") {
		t.Error("expected target doc ID in prompt")
	}
}

func TestBuild_CustomDirectives(t *testing.T) {
	b := NewBuilder()

	state := domain.DefaultStateData()
	state.CustomDirectives["format"] = "always use bullet points"
	state.UserDisplayName = "Budi"

	sc := SessionContext{
		State:   &state,
		Persona: Persona{Name: "A", Description: "B"},
	}

	result := b.Build(sc)

	if !strings.Contains(result, "bullet points") {
		t.Error("expected custom directive in prompt")
	}
	if !strings.Contains(result, "Budi") {
		t.Error("expected user display name in prompt")
	}
}

func TestBuild_ConciseVerbosity(t *testing.T) {
	b := NewBuilder()

	state := domain.DefaultStateData()
	state.Verbosity = "concise"

	sc := SessionContext{
		State:   &state,
		Persona: Persona{Name: "A", Description: "B"},
	}

	result := b.Build(sc)

	if !strings.Contains(result, "SINGKAT") {
		t.Error("expected concise verbosity rule")
	}
}

func TestBuild_WithTemplate(t *testing.T) {
	b := NewBuilder()

	state := domain.DefaultStateData()
	sc := SessionContext{
		State:          &state,
		Persona:        Persona{Name: "Pribadi", Description: "Smart AI", Tone: "formal"},
		PromptTemplate: "Anda adalah {{persona_name}}. {{persona_description}}. Gaya: {{tone_preference}}.\n{{behavioral_rules}}\nFakta: {{user_facts}}\nDokumen: {{active_document_metadata}}\nRAG: {{rag_context}}",
		MemoryContext:  "User suka kopi",
	}

	result := b.Build(sc)

	if !strings.Contains(result, "Pribadi") {
		t.Error("expected persona name substituted")
	}
	if !strings.Contains(result, "Smart AI") {
		t.Error("expected description substituted")
	}
	if !strings.Contains(result, "ATURAN PERILAKU") {
		t.Error("expected behavioral rules injected via template")
	}
}

func TestTruncateRAG(t *testing.T) {
	b := NewBuilder()

	long := strings.Repeat("x", 5000)
	result := b.truncateRAG(long)

	if len(result) > maxContextChars+50 {
		t.Errorf("expected truncation, got len=%d", len(result))
	}
	if !strings.Contains(result, "[...context truncated...]") {
		t.Error("expected truncation marker")
	}
}
