package response

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/achmichael/pribadi-go/internal/classifier"
	"github.com/achmichael/pribadi-go/internal/conversation"
	"github.com/achmichael/pribadi-go/internal/domain"
	"github.com/achmichael/pribadi-go/internal/prompt"
	"github.com/achmichael/pribadi-go/internal/reasoning"
	"github.com/achmichael/pribadi-go/internal/repository"
	"github.com/rs/zerolog"
)

func setupTestRepo(t *testing.T) repository.Repository {
	t.Helper()
	tmpFile, err := os.CreateTemp("", "test-response-*.db")
	if err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()
	t.Cleanup(func() { os.Remove(tmpFile.Name()) })

	repo, err := repository.NewSQLiteRepository(tmpFile.Name(), "../../db/schema.sql")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { repo.Close() })
	return repo
}

func TestProcessor_BasicProcess(t *testing.T) {
	logger := zerolog.New(os.Stderr).With().Logger()
	repo := setupTestRepo(t)
	sm := conversation.NewStateManager(repo, &logger)
	proc := NewProcessor(sm, nil, repo, &logger)

	state := domain.DefaultStateData()
	params := ProcessParams{
		UserID:      "user-1",
		SessionID:   "default",
		UserMessage: "What is AI?",
		RawResponse: "AI is artificial intelligence.",
		State:       &state,
		RAGSources:  []string{"doc1.pdf", "doc2.pdf"},
		ComposedPrompt: &prompt.ComposedPrompt{
			IncludedRAG:    true,
			IncludedMemory: true,
		},
		Verification: &reasoning.VerificationResult{
			IsValid:           true,
			ConfidenceScore:   0.9,
			HallucinationRisk: "low",
		},
	}

	result, err := proc.Process(context.Background(), params)
	if err != nil {
		t.Fatal(err)
	}

	if result.Text == "" {
		t.Error("expected non-empty response text")
	}
	if len(result.Citations) != 2 {
		t.Errorf("expected 2 citations, got %d", len(result.Citations))
	}
	if result.Metadata.ConfidenceScore != 0.9 {
		t.Errorf("expected confidence=0.9, got %f", result.Metadata.ConfidenceScore)
	}
	if !result.Metadata.RAGUsed {
		t.Error("expected RAGUsed=true")
	}
}

func TestProcessor_NoCitations(t *testing.T) {
	logger := zerolog.New(os.Stderr).With().Logger()
	repo := setupTestRepo(t)
	sm := conversation.NewStateManager(repo, &logger)
	proc := NewProcessor(sm, nil, repo, &logger)

	state := domain.DefaultStateData()
	params := ProcessParams{
		UserID:      "user-2",
		SessionID:   "default",
		UserMessage: "Hello",
		RawResponse: "Hi there!",
		State:       &state,
		RAGSources:  nil, // no sources
	}

	result, err := proc.Process(context.Background(), params)
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Citations) > 0 {
		t.Error("expected no citations")
	}
	if result.Text != "Hi there!" {
		t.Errorf("expected unchanged response, got %s", result.Text)
	}
}

func TestProcessor_ShouldNotUpdate_SimpleGreeting(t *testing.T) {
	logger := zerolog.New(os.Stderr).With().Logger()
	repo := setupTestRepo(t)
	sm := conversation.NewStateManager(repo, &logger)
	proc := NewProcessor(sm, nil, repo, &logger)

	state := domain.DefaultStateData()
	params := ProcessParams{
		UserID:      "user-3",
		SessionID:   "default",
		UserMessage: "Hello",
		RawResponse: "Hi!",
		State:       &state,
		Classification: &classifier.Classification{
			Intent: classifier.IntentChitChat,
		},
	}

	result, err := proc.Process(context.Background(), params)
	if err != nil {
		t.Fatal(err)
	}

	if result.ShouldUpdate {
		t.Error("expected ShouldUpdate=false for simple greeting")
	}
}

func TestProcessor_UpdateState(t *testing.T) {
	logger := zerolog.New(os.Stderr).With().Logger()
	repo := setupTestRepo(t)
	sm := conversation.NewStateManager(repo, &logger)
	proc := NewProcessor(sm, nil, repo, &logger)

	// Create user
	repo.CreateUser(context.Background(), "user-4", "test")

	class := &classifier.Classification{
		MessageClass: classifier.ClassQuestion,
		Intent:       classifier.IntentAskInfo,
	}

	err := proc.UpdateState(context.Background(), "user-4", "default", class)
	if err != nil {
		t.Fatal(err)
	}

	// Verify state was updated (check turn count incremented)
	rawState, err := sm.GetRaw(context.Background(), "user-4", "default")
	if err != nil {
		t.Fatal(err)
	}
	if rawState.LastIntent != string(classifier.IntentAskInfo) {
		t.Errorf("expected intent=%s, got %s", classifier.IntentAskInfo, rawState.LastIntent)
	}
}

func TestProcessor_DetermineCitationStyle(t *testing.T) {
	logger := zerolog.New(os.Stderr).With().Logger()
	repo := setupTestRepo(t)
	proc := &processor{
		repo:   repo,
		logger: &logger,
	}

	// Default
	style := proc.determineCitationStyle(nil)
	if style != prompt.CitationInline {
		t.Errorf("expected inline by default, got %s", style)
	}

	// Custom directive: footnote
	state := domain.DefaultStateData()
	state.CustomDirectives = map[string]string{
		"citation_style": "footnote",
	}
	style = proc.determineCitationStyle(&state)
	if style != prompt.CitationFootnote {
		t.Errorf("expected footnote, got %s", style)
	}

	// Custom directive: none
	state.CustomDirectives["citation_style"] = "none"
	style = proc.determineCitationStyle(&state)
	if style != prompt.CitationNone {
		t.Errorf("expected none, got %s", style)
	}
}

func TestInteractionLogger_Log(t *testing.T) {
	logger := zerolog.New(os.Stderr).With().Logger()
	repo := setupTestRepo(t)
	iLogger := NewInteractionLogger(repo, &logger)

	log := InteractionLog{
		UserID:      "user-5",
		SessionID:   "default",
		Timestamp:   time.Now(),
		UserMessage: "test",
		Response:    "response",
		Metadata: map[string]interface{}{
			"confidence": 0.8,
		},
		Duration: 100,
		Success:  true,
	}

	err := iLogger.Log(context.Background(), log)
	if err != nil {
		t.Fatal(err)
	}
}

func TestInteractionLogger_LogError(t *testing.T) {
	logger := zerolog.New(os.Stderr).With().Logger()
	repo := setupTestRepo(t)
	iLogger := NewInteractionLogger(repo, &logger)

	err := iLogger.LogError(context.Background(), "user-6", "default", "test", "some error", 50)
	if err != nil {
		t.Fatal(err)
	}
}

func TestMetricsAggregator_Empty(t *testing.T) {
	agg := NewMetricsAggregator()
	metrics := agg.Aggregate(nil)

	if metrics.TotalInteractions != 0 {
		t.Error("expected 0 interactions")
	}
}

func TestMetricsAggregator_Basic(t *testing.T) {
	agg := NewMetricsAggregator()

	logs := []InteractionLog{
		{
			UserID:   "user-1",
			Duration: 100,
			Success:  true,
			Metadata: map[string]interface{}{
				"confidence_score":   0.9,
				"hallucination_risk": "low",
				"intent":             "ask_information",
				"strategy":           "direct",
			},
		},
		{
			UserID:   "user-2",
			Duration: 200,
			Success:  true,
			Metadata: map[string]interface{}{
				"confidence_score":   0.7,
				"hallucination_risk": "high",
				"intent":             "ask_information",
				"strategy":           "step_by_step",
			},
		},
		{
			UserID:   "user-3",
			Duration: 150,
			Success:  false,
			Metadata: map[string]interface{}{},
		},
	}

	metrics := agg.Aggregate(logs)

	if metrics.TotalInteractions != 3 {
		t.Errorf("expected 3 interactions, got %d", metrics.TotalInteractions)
	}
	if metrics.SuccessRate < 0.66 || metrics.SuccessRate > 0.67 {
		t.Errorf("expected success rate ~0.67, got %f", metrics.SuccessRate)
	}
	if metrics.AvgDurationMs != 150 {
		t.Errorf("expected avg duration=150, got %f", metrics.AvgDurationMs)
	}
	if metrics.HighRiskCount != 1 {
		t.Errorf("expected 1 high risk, got %d", metrics.HighRiskCount)
	}
	if metrics.IntentBreakdown["ask_information"] != 2 {
		t.Errorf("expected 2 ask_information, got %d", metrics.IntentBreakdown["ask_information"])
	}
	if metrics.StrategyBreakdown["direct"] != 1 {
		t.Errorf("expected 1 direct, got %d", metrics.StrategyBreakdown["direct"])
	}
}

func TestProcessor_BuildCitations(t *testing.T) {
	logger := zerolog.New(os.Stderr).With().Logger()
	proc := &processor{logger: &logger}

	sources := []string{"doc1.pdf", "doc2.pdf", "doc3.pdf"}
	citations := proc.buildCitations(sources)

	if len(citations) != 3 {
		t.Fatalf("expected 3 citations, got %d", len(citations))
	}
	if citations[0].Number != 1 {
		t.Errorf("expected first citation number=1, got %d", citations[0].Number)
	}
	if citations[1].Source != "doc2.pdf" {
		t.Errorf("expected second source=doc2.pdf, got %s", citations[1].Source)
	}
	if citations[2].Type != "document" {
		t.Errorf("expected type=document, got %s", citations[2].Type)
	}
}
