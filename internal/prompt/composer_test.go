package prompt

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/achmichael/pribadi-go/internal/classifier"
	"github.com/achmichael/pribadi-go/internal/domain"
	"github.com/achmichael/pribadi-go/internal/planner"
	"github.com/rs/zerolog"
)

func TestComposer_BasicCompose(t *testing.T) {
	logger := zerolog.New(os.Stderr).With().Logger()
	builder := NewBuilder()
	composer := NewComposer(builder, &logger)

	state := domain.DefaultStateData()
	sc := &SessionContext{
		State: &state,
		Persona: Persona{
			Name:        "TestBot",
			Description: "Test assistant",
			Tone:        "casual",
		},
	}

	result, err := composer.Compose(context.Background(), ComposeParams{
		UserText:       "Apa itu AI?",
		SessionContext: sc,
	})

	if err != nil {
		t.Fatal(err)
	}
	if result.SystemPrompt == "" {
		t.Error("expected non-empty system prompt")
	}
	if result.UserPrompt != "Apa itu AI?" {
		t.Errorf("expected user prompt unchanged, got %s", result.UserPrompt)
	}
}

func TestComposer_WithCorrection(t *testing.T) {
	logger := zerolog.New(os.Stderr).With().Logger()
	builder := NewBuilder()
	composer := NewComposer(builder, &logger)

	state := domain.DefaultStateData()
	sc := &SessionContext{
		State: &state,
		Persona: Persona{
			Name: "TestBot",
		},
	}

	class := &classifier.Classification{
		Intent:       classifier.IntentCorrect,
		IsCorrection: true,
	}

	result, err := composer.Compose(context.Background(), ComposeParams{
		UserText:       "Bukan, yang benar adalah X",
		Classification: class,
		SessionContext: sc,
	})

	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.SystemPrompt, "KOREKSI") {
		t.Error("expected correction directive in system prompt")
	}
	if result.Intent != string(classifier.IntentCorrect) {
		t.Errorf("expected intent=correct, got %s", result.Intent)
	}
}

func TestComposer_WithStepByStepPlan(t *testing.T) {
	logger := zerolog.New(os.Stderr).With().Logger()
	builder := NewBuilder()
	composer := NewComposer(builder, &logger)

	state := domain.DefaultStateData()
	sc := &SessionContext{
		State:   &state,
		Persona: Persona{Name: "Bot"},
	}

	plan := &planner.Plan{
		ResponseStrategy: "step_by_step",
		NeedsRAG:         true,
	}

	result, err := composer.Compose(context.Background(), ComposeParams{
		UserText:       "Bagaimana cara membuat kue?",
		Plan:           plan,
		SessionContext: sc,
	})

	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.SystemPrompt, "step-by-step") {
		t.Error("expected step-by-step directive")
	}
}

func TestComposer_WithClarificationNeeded(t *testing.T) {
	logger := zerolog.New(os.Stderr).With().Logger()
	builder := NewBuilder()
	composer := NewComposer(builder, &logger)

	state := domain.DefaultStateData()
	sc := &SessionContext{
		State:   &state,
		Persona: Persona{Name: "Bot"},
	}

	class := &classifier.Classification{
		RequiresClarify: true,
	}

	result, err := composer.Compose(context.Background(), ComposeParams{
		UserText:       "ambiguous question",
		Classification: class,
		SessionContext: sc,
	})

	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.SystemPrompt, "AMBIGUITAS") {
		t.Error("expected ambiguity warning in system prompt")
	}
}

func TestComposer_ApplyPlanFilters_SkipRAG(t *testing.T) {
	logger := zerolog.New(os.Stderr).With().Logger()
	composer := &composer{logger: &logger}

	state := domain.DefaultStateData()
	sc := &SessionContext{
		State:      &state,
		HasRAG:     true,
		RAGContext: "some rag content",
		RAGSources: []string{"doc1", "doc2"},
	}

	plan := &planner.Plan{
		NeedsRAG: false,
	}

	filtered := composer.applyPlanFilters(sc, plan)

	if filtered.HasRAG {
		t.Error("expected HasRAG=false after filtering")
	}
	if filtered.RAGContext != "" {
		t.Error("expected RAG context cleared")
	}
	if len(filtered.RAGSources) > 0 {
		t.Error("expected RAG sources cleared")
	}
}

func TestComposer_ApplyPlanFilters_SkipMemory(t *testing.T) {
	logger := zerolog.New(os.Stderr).With().Logger()
	composer := &composer{logger: &logger}

	state := domain.DefaultStateData()
	sc := &SessionContext{
		State:         &state,
		MemoryContext: "user facts...",
	}

	plan := &planner.Plan{
		NeedsMemory: false,
	}

	filtered := composer.applyPlanFilters(sc, plan)

	if filtered.MemoryContext != "" {
		t.Error("expected memory context cleared")
	}
}

func TestComposer_WithConversationHistory(t *testing.T) {
	logger := zerolog.New(os.Stderr).With().Logger()
	builder := NewBuilder()
	composer := NewComposer(builder, &logger)

	state := domain.DefaultStateData()
	sc := &SessionContext{
		State:   &state,
		Persona: Persona{Name: "Bot"},
	}

	history := []HistoryMessage{
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi there!"},
	}

	result, err := composer.Compose(context.Background(), ComposeParams{
		UserText:            "Continue from before",
		SessionContext:      sc,
		ConversationHistory: history,
	})

	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.UserPrompt, "Recent Conversation History") {
		t.Error("expected history in user prompt")
	}
	if !strings.Contains(result.UserPrompt, "Hello") {
		t.Error("expected history content in user prompt")
	}
	if !result.IncludedHistory {
		t.Error("expected IncludedHistory=true")
	}
}

func TestComposer_NilSessionContext(t *testing.T) {
	logger := zerolog.New(os.Stderr).With().Logger()
	builder := NewBuilder()
	composer := NewComposer(builder, &logger)

	_, err := composer.Compose(context.Background(), ComposeParams{
		UserText:       "test",
		SessionContext: nil,
	})

	if err == nil {
		t.Error("expected error with nil session context")
	}
}
