package planner

import (
	"context"
	"os"
	"testing"

	"github.com/achmichael/pribadi-go/internal/classifier"
	"github.com/achmichael/pribadi-go/internal/domain"
	"github.com/rs/zerolog"
)

func TestHeuristicPlan_ChitChat(t *testing.T) {
	logger := zerolog.New(os.Stderr).With().Logger()
	p := &planner{logger: &logger}

	plan := p.heuristicPlan(PlanParams{
		Classification: &classifier.Classification{
			Intent: classifier.IntentChitChat,
		},
	})

	if plan == nil {
		t.Fatal("expected plan, got nil")
	}
	if plan.NeedsRAG {
		t.Error("chitchat shouldn't need RAG")
	}
	if !plan.NeedsMemory {
		t.Error("chitchat should use memory for personalization")
	}
	if plan.ResponseStrategy != "direct" {
		t.Errorf("expected direct strategy, got %s", plan.ResponseStrategy)
	}
}

func TestHeuristicPlan_Preference(t *testing.T) {
	logger := zerolog.New(os.Stderr).With().Logger()
	p := &planner{logger: &logger}

	plan := p.heuristicPlan(PlanParams{
		Classification: &classifier.Classification{
			Intent: classifier.IntentSetPreference,
		},
	})

	if plan == nil {
		t.Fatal("expected plan, got nil")
	}
	if plan.NeedsRAG || plan.NeedsMemory || plan.NeedsHistory {
		t.Error("preference update shouldn't need any retrieval")
	}
}

func TestHeuristicPlan_DocumentQuestion(t *testing.T) {
	logger := zerolog.New(os.Stderr).With().Logger()
	p := &planner{logger: &logger}

	plan := p.heuristicPlan(PlanParams{
		Classification: &classifier.Classification{
			Intent: classifier.IntentAskDocument,
		},
	})

	if plan == nil {
		t.Fatal("expected plan, got nil")
	}
	if !plan.NeedsRAG {
		t.Error("document question needs RAG")
	}
	if !plan.NeedsDocContext {
		t.Error("document question needs doc context")
	}
}

func TestHeuristicPlan_Correction(t *testing.T) {
	logger := zerolog.New(os.Stderr).With().Logger()
	p := &planner{logger: &logger}

	plan := p.heuristicPlan(PlanParams{
		Classification: &classifier.Classification{
			Intent: classifier.IntentCorrect,
		},
		HasActiveDoc: true,
	})

	if plan == nil {
		t.Fatal("expected plan, got nil")
	}
	if !plan.NeedsHistory {
		t.Error("correction needs history")
	}
	if !plan.NeedsDocContext {
		t.Error("correction with active doc needs doc context")
	}
}

func TestHeuristicPlan_NoMatch(t *testing.T) {
	logger := zerolog.New(os.Stderr).With().Logger()
	p := &planner{logger: &logger}

	plan := p.heuristicPlan(PlanParams{
		Classification: &classifier.Classification{
			Intent: classifier.IntentAskInfo,
		},
	})

	if plan != nil {
		t.Error("expected nil for non-heuristic intent")
	}
}

func TestFallbackPlan(t *testing.T) {
	logger := zerolog.New(os.Stderr).With().Logger()
	p := &planner{logger: &logger}

	plan := p.fallbackPlan(PlanParams{HasActiveDoc: true})

	if !plan.NeedsRAG || !plan.NeedsMemory || !plan.NeedsHistory {
		t.Error("fallback should retrieve everything")
	}
	if !plan.NeedsDocContext {
		t.Error("fallback with active doc should include doc context")
	}
	if plan.Confidence != "low" {
		t.Errorf("fallback confidence should be low, got %s", plan.Confidence)
	}
}

func TestCreatePlan_HeuristicPath(t *testing.T) {
	logger := zerolog.New(os.Stderr).With().Logger()
	p := NewPlanner(nil, &logger) // nil LLM — will use heuristic

	state := domain.DefaultStateData()
	plan, err := p.CreatePlan(context.Background(), PlanParams{
		UserText: "halo",
		Classification: &classifier.Classification{
			Intent: classifier.IntentChitChat,
		},
		State: &state,
	})

	if err != nil {
		t.Fatal(err)
	}
	if plan.ResponseStrategy != "direct" {
		t.Errorf("expected direct, got %s", plan.ResponseStrategy)
	}
}
