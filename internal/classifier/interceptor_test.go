package classifier

import (
	"context"
	"os"
	"testing"

	"github.com/achmichael/pribadi-go/internal/conversation"
	"github.com/achmichael/pribadi-go/internal/domain"
	"github.com/achmichael/pribadi-go/internal/repository"
	"github.com/rs/zerolog"
)

func setupTestRepo(t *testing.T) repository.Repository {
	t.Helper()
	tmpFile, err := os.CreateTemp("", "test-interceptor-*.db")
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

func TestInterceptor_PreferenceUpdate(t *testing.T) {
	repo := setupTestRepo(t)
	logger := zerolog.New(os.Stderr).With().Logger()

	sm := conversation.NewStateManager(repo, &logger)
	pm := conversation.NewPreferenceManager(repo, &logger)
	ic := NewInterceptor(sm, pm, &logger, nil, nil)

	// Create test user
	repo.CreateUser(context.Background(), "user-i1", "test")

	state := domain.DefaultStateData()
	class := &Classification{
		MessageClass:  ClassPreference,
		Intent:        IntentSetPreference,
		IsInstruction: true,
		ExtractedPrefs: []PrefUpdate{
			{Key: "language", Value: "en"},
			{Key: "display_name", Value: "Budi"},
		},
	}

	intercepted, ack, err := ic.Process(context.Background(), InterceptParams{
		UserID:    "user-i1",
		SessionID: "default",
		UserText:  "Jawab dalam bahasa Inggris, panggil aku Budi",
		Class:     class,
		State:     &state,
	})

	if err != nil {
		t.Fatal(err)
	}
	if !intercepted {
		t.Error("expected intercepted=true")
	}
	if ack == "" {
		t.Error("expected non-empty ack")
	}

	// Verify state was updated
	loaded, err := sm.Load(context.Background(), "user-i1", "default")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Language != "en" {
		t.Errorf("expected language=en in state, got=%s", loaded.Language)
	}
	if loaded.UserDisplayName != "Budi" {
		t.Errorf("expected name=Budi in state, got=%s", loaded.UserDisplayName)
	}

	// Verify preference was persisted
	pref, _ := pm.Get(context.Background(), "user-i1", "language")
	if pref == nil || pref.PrefValue != "en" {
		t.Error("expected persistent pref language=en")
	}
}

func TestInterceptor_CorrectionNotIntercepted(t *testing.T) {
	repo := setupTestRepo(t)
	logger := zerolog.New(os.Stderr).With().Logger()

	sm := conversation.NewStateManager(repo, &logger)
	pm := conversation.NewPreferenceManager(repo, &logger)
	ic := NewInterceptor(sm, pm, &logger, nil, nil)

	repo.CreateUser(context.Background(), "user-i2", "test")

	state := domain.DefaultStateData()
	class := &Classification{
		MessageClass: ClassCorrection,
		Intent:       IntentCorrect,
		IsCorrection: true,
	}

	intercepted, _, err := ic.Process(context.Background(), InterceptParams{
		UserID:    "user-i2",
		SessionID: "default",
		UserText:  "Bukan, yang benar adalah Python bukan Java",
		Class:     class,
		State:     &state,
	})

	if err != nil {
		t.Fatal(err)
	}
	// Corrections update state but DON'T intercept (still need LLM response)
	if intercepted {
		t.Error("expected intercepted=false for correction")
	}

	// But state should be updated
	loaded, _ := sm.Load(context.Background(), "user-i2", "default")
	if loaded.LastCorrectionAt == "" {
		t.Error("expected last_correction_at to be set")
	}
}

func TestInterceptor_NilClass(t *testing.T) {
	repo := setupTestRepo(t)
	logger := zerolog.New(os.Stderr).With().Logger()

	sm := conversation.NewStateManager(repo, &logger)
	pm := conversation.NewPreferenceManager(repo, &logger)
	ic := NewInterceptor(sm, pm, &logger, nil, nil)

	intercepted, _, err := ic.Process(context.Background(), InterceptParams{
		Class: nil,
	})

	if err != nil {
		t.Fatal(err)
	}
	if intercepted {
		t.Error("expected not intercepted when class is nil")
	}
}

func TestInterceptor_NormalQuestionNotIntercepted(t *testing.T) {
	repo := setupTestRepo(t)
	logger := zerolog.New(os.Stderr).With().Logger()

	sm := conversation.NewStateManager(repo, &logger)
	pm := conversation.NewPreferenceManager(repo, &logger)
	ic := NewInterceptor(sm, pm, &logger, nil, nil)

	class := &Classification{
		MessageClass: ClassQuestion,
		Intent:       IntentAskInfo,
	}

	intercepted, _, err := ic.Process(context.Background(), InterceptParams{
		UserText: "Apa itu machine learning?",
		Class:    class,
	})

	if err != nil {
		t.Fatal(err)
	}
	if intercepted {
		t.Error("expected not intercepted for normal question")
	}
}
