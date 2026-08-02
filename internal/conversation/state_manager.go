// Package conversation manages ephemeral per-session conversation state
// and persistent per-user preferences. It is the single source of truth
// for "how should the AI behave right now" for a given user+session.
package conversation

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/achmichael/pribadi-go/internal/domain"
	"github.com/achmichael/pribadi-go/internal/repository"
	"github.com/rs/zerolog"
)

// ─── Interface ─────────────────────────────────────────────────────

// StateManager loads, updates, and persists conversation state.
type StateManager interface {
	// Load returns the current state for a user+session.
	// Creates default state if none exists.
	Load(ctx context.Context, userID, sessionID string) (*domain.StateData, error)

	// Update persists the full state + classification metadata.
	Update(ctx context.Context, userID, sessionID string, state *domain.StateData, intent, messageClass, activeTask string) error

	// GetRaw returns the raw DB row (for turn_count, timestamps, etc.).
	GetRaw(ctx context.Context, userID, sessionID string) (*repository.ConversationStateRow, error)

	DeleteState(ctx context.Context, userID, sessionID string) error

	// ApplyPreferences merges persistent user preferences into state.
	// Preferences are lower priority than explicit session state.
	ApplyPreferences(ctx context.Context, userID string, state *domain.StateData) error
}

// ─── Implementation ────────────────────────────────────────────────

type stateManager struct {
	repo   repository.Repository
	logger *zerolog.Logger
	mu     sync.Mutex // serialize writes for same user
}

// NewStateManager creates a StateManager backed by SQLite.
func NewStateManager(repo repository.Repository, logger *zerolog.Logger) StateManager {
	return &stateManager{
		repo:   repo,
		logger: logger,
	}
}

func (m *stateManager) DeleteState(ctx context.Context, userID, sessionID string) error {
	return m.repo.DeleteConversationState(ctx, userID, sessionID)
}

func (m *stateManager) Load(ctx context.Context, userID, sessionID string) (*domain.StateData, error) {
	row, err := m.repo.GetConversationState(ctx, userID, sessionID)
	if err != nil {
		return nil, fmt.Errorf("load conversation state: %w", err)
	}

	if row == nil {
		// First interaction in this session — create default state
		state := domain.DefaultStateData()

		// Overlay persistent preferences
		if err := m.ApplyPreferences(ctx, userID, &state); err != nil {
			m.logger.Warn().Err(err).
				Str("user_id", userID).
				Msg("[state] failed to apply preferences on init")
		}

		m.logger.Info().
			Str("user_id", userID).
			Str("session_id", sessionID).
			Msg("[state] initialized default state")

		return &state, nil
	}

	var state domain.StateData
	if err := json.Unmarshal([]byte(row.StateJSON), &state); err != nil {
		m.logger.Warn().Err(err).
			Str("user_id", userID).
			Str("raw", row.StateJSON).
			Msg("[state] corrupt state_json, resetting to default")
		state = domain.DefaultStateData()
	}

	// Ensure map is initialized
	if state.CustomDirectives == nil {
		state.CustomDirectives = make(map[string]string)
	}

	return &state, nil
}

func (m *stateManager) Update(ctx context.Context, userID, sessionID string, state *domain.StateData, intent, messageClass, activeTask string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	stateBytes, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}

	err = m.repo.UpsertConversationState(ctx, repository.UpsertConversationStateParams{
		UserID:           userID,
		SessionID:        sessionID,
		StateJSON:        string(stateBytes),
		LastIntent:       intent,
		LastMessageClass: messageClass,
		ActiveTask:       activeTask,
	})
	if err != nil {
		return fmt.Errorf("upsert conversation state: %w", err)
	}

	m.logger.Debug().
		Str("user_id", userID).
		Str("session_id", sessionID).
		Str("intent", intent).
		Str("class", messageClass).
		Msg("[state] updated")

	return nil
}

func (m *stateManager) GetRaw(ctx context.Context, userID, sessionID string) (*repository.ConversationStateRow, error) {
	return m.repo.GetConversationState(ctx, userID, sessionID)
}

func (m *stateManager) ApplyPreferences(ctx context.Context, userID string, state *domain.StateData) error {
	prefs, err := m.repo.ListUserPreferences(ctx, userID)
	if err != nil {
		return fmt.Errorf("list user preferences: %w", err)
	}

	for _, p := range prefs {
		// Only apply if state field is still at default (don't override session-level changes)
		switch p.PrefKey {
		case domain.PrefKeyLanguage:
			if state.Language == "id" { // default
				state.Language = p.PrefValue
			}
		case domain.PrefKeyTone:
			if state.Tone == "casual" { // default
				state.Tone = p.PrefValue
			}
		case domain.PrefKeyVerbosity:
			if state.Verbosity == "normal" { // default
				state.Verbosity = p.PrefValue
			}
		case domain.PrefKeyResponseFormat:
			if state.ResponseFormat == "plain" { // default
				state.ResponseFormat = p.PrefValue
			}
		case domain.PrefKeyDisplayName:
			if state.UserDisplayName == "" { // default
				state.UserDisplayName = p.PrefValue
			}
		}
	}

	return nil
}
