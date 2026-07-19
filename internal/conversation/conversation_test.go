package conversation

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/achmichael/pribadi-go/internal/domain"
	"github.com/achmichael/pribadi-go/internal/repository"
	"github.com/rs/zerolog"
)

func setupTestDB(t *testing.T) repository.Repository {
	t.Helper()
	tmpFile, err := os.CreateTemp("", "test-*.db")
	if err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()
	t.Cleanup(func() { os.Remove(tmpFile.Name()) })

	// We need the schema + migrations
	repo, err := repository.NewSQLiteRepository(tmpFile.Name(), "../../db/schema.sql")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { repo.Close() })
	return repo
}

func createTestUser(t *testing.T, repo repository.Repository, userID string) {
	t.Helper()
	if err := repo.CreateUser(context.Background(), userID, "test"); err != nil {
		t.Fatal(err)
	}
}

func TestStateManager_LoadDefault(t *testing.T) {
	repo := setupTestDB(t)
	logger := zerolog.New(os.Stderr).With().Logger()
	sm := NewStateManager(repo, &logger)

	createTestUser(t, repo, "user-1")

	state, err := sm.Load(context.Background(), "user-1", "default")
	if err != nil {
		t.Fatal(err)
	}

	if state.Language != "id" {
		t.Errorf("expected language=id, got %s", state.Language)
	}
	if state.Tone != "casual" {
		t.Errorf("expected tone=casual, got %s", state.Tone)
	}
	if state.Verbosity != "normal" {
		t.Errorf("expected verbosity=normal, got %s", state.Verbosity)
	}
}

func TestStateManager_UpdateAndLoad(t *testing.T) {
	repo := setupTestDB(t)
	logger := zerolog.New(os.Stderr).With().Logger()
	sm := NewStateManager(repo, &logger)

	createTestUser(t, repo, "user-2")

	state := domain.DefaultStateData()
	state.Language = "en"
	state.Tone = "formal"
	state.UserDisplayName = "Michael"

	err := sm.Update(context.Background(), "user-2", "sess-1", &state, "question", "question", "")
	if err != nil {
		t.Fatal(err)
	}

	loaded, err := sm.Load(context.Background(), "user-2", "sess-1")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Language != "en" {
		t.Errorf("expected language=en, got %s", loaded.Language)
	}
	if loaded.Tone != "formal" {
		t.Errorf("expected tone=formal, got %s", loaded.Tone)
	}
	if loaded.UserDisplayName != "Michael" {
		t.Errorf("expected name=Michael, got %s", loaded.UserDisplayName)
	}
}

func TestStateManager_TurnCount(t *testing.T) {
	repo := setupTestDB(t)
	logger := zerolog.New(os.Stderr).With().Logger()
	sm := NewStateManager(repo, &logger)

	createTestUser(t, repo, "user-3")

	state := domain.DefaultStateData()

	// First update = turn 1
	sm.Update(context.Background(), "user-3", "default", &state, "q", "question", "")
	// Second update = turn 2
	sm.Update(context.Background(), "user-3", "default", &state, "q", "question", "")

	raw, err := sm.GetRaw(context.Background(), "user-3", "default")
	if err != nil {
		t.Fatal(err)
	}
	if raw == nil {
		t.Fatal("expected raw state, got nil")
	}
	if raw.TurnCount != 2 {
		t.Errorf("expected turn_count=2, got %d", raw.TurnCount)
	}
}

func TestPreferenceManager_SetAndGet(t *testing.T) {
	repo := setupTestDB(t)
	logger := zerolog.New(os.Stderr).With().Logger()
	pm := NewPreferenceManager(repo, &logger)

	createTestUser(t, repo, "user-4")

	err := pm.SetExplicit(context.Background(), "user-4", domain.PrefKeyLanguage, "en")
	if err != nil {
		t.Fatal(err)
	}

	pref, err := pm.Get(context.Background(), "user-4", domain.PrefKeyLanguage)
	if err != nil {
		t.Fatal(err)
	}
	if pref == nil {
		t.Fatal("expected pref, got nil")
	}
	if pref.PrefValue != "en" {
		t.Errorf("expected en, got %s", pref.PrefValue)
	}
	if pref.Source != "explicit" {
		t.Errorf("expected source=explicit, got %s", pref.Source)
	}
	if pref.Confidence != "high" {
		t.Errorf("expected confidence=high, got %s", pref.Confidence)
	}
}

func TestPreferenceManager_LoadDefaults(t *testing.T) {
	repo := setupTestDB(t)
	logger := zerolog.New(os.Stderr).With().Logger()
	pm := NewPreferenceManager(repo, &logger)

	createTestUser(t, repo, "user-5")

	err := pm.LoadDefaults(context.Background(), "user-5")
	if err != nil {
		t.Fatal(err)
	}

	prefs, err := pm.GetAll(context.Background(), "user-5")
	if err != nil {
		t.Fatal(err)
	}

	if len(prefs) != 4 {
		t.Errorf("expected 4 defaults, got %d", len(prefs))
	}
}

func TestPreferenceManager_ExplicitOverridesDefault(t *testing.T) {
	repo := setupTestDB(t)
	logger := zerolog.New(os.Stderr).With().Logger()
	pm := NewPreferenceManager(repo, &logger)

	createTestUser(t, repo, "user-6")

	// Set default first
	pm.LoadDefaults(context.Background(), "user-6")

	// Override with explicit
	pm.SetExplicit(context.Background(), "user-6", domain.PrefKeyLanguage, "en")

	pref, _ := pm.Get(context.Background(), "user-6", domain.PrefKeyLanguage)
	if pref.PrefValue != "en" {
		t.Errorf("expected en, got %s", pref.PrefValue)
	}
	if pref.Source != "explicit" {
		t.Errorf("expected source=explicit after override, got %s", pref.Source)
	}
}

func TestStateManager_ApplyPreferences(t *testing.T) {
	repo := setupTestDB(t)
	logger := zerolog.New(os.Stderr).With().Logger()
	sm := NewStateManager(repo, &logger)
	pm := NewPreferenceManager(repo, &logger)

	createTestUser(t, repo, "user-7")

	// Set explicit prefs
	pm.SetExplicit(context.Background(), "user-7", domain.PrefKeyLanguage, "en")
	pm.SetExplicit(context.Background(), "user-7", domain.PrefKeyDisplayName, "Budi")

	// Load default state and apply prefs
	state := domain.DefaultStateData()
	err := sm.ApplyPreferences(context.Background(), "user-7", &state)
	if err != nil {
		t.Fatal(err)
	}

	if state.Language != "en" {
		t.Errorf("expected language=en after pref apply, got %s", state.Language)
	}
	if state.UserDisplayName != "Budi" {
		t.Errorf("expected name=Budi after pref apply, got %s", state.UserDisplayName)
	}
}

// Suppress unused import warning
var _ = sql.ErrNoRows
