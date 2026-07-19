package conversation

import (
	"context"
	"fmt"

	"github.com/achmichael/pribadi-go/internal/domain"
	"github.com/achmichael/pribadi-go/internal/repository"
	"github.com/rs/zerolog"
)

// ─── Interface ─────────────────────────────────────────────────────

// PreferenceManager handles persistent per-user preferences.
// These survive across sessions and are applied as defaults when
// a new session starts (lower priority than session-level state).
type PreferenceManager interface {
	// Get returns a single preference. Returns nil if not set.
	Get(ctx context.Context, userID, key string) (*repository.UserPreferenceRow, error)

	// GetAll returns all preferences for a user.
	GetAll(ctx context.Context, userID string) ([]repository.UserPreferenceRow, error)

	// Set persists a preference with source and confidence metadata.
	Set(ctx context.Context, userID, key, value, source, confidence string) error

	// SetExplicit is a shorthand for Set with source="explicit", confidence="high".
	// Used when user directly states a preference.
	SetExplicit(ctx context.Context, userID, key, value string) error

	// SetInferred is a shorthand for Set with source="inferred", confidence="medium".
	// Used when preference is derived from behavior.
	SetInferred(ctx context.Context, userID, key, value string) error

	// Delete removes a preference.
	Delete(ctx context.Context, userID, key string) error

	// LoadDefaults seeds default preferences for a new user if none exist.
	LoadDefaults(ctx context.Context, userID string) error

	// ToStateOverrides converts all preferences to a map for prompt building.
	ToStateOverrides(ctx context.Context, userID string) (map[string]string, error)
}

// ─── Implementation ────────────────────────────────────────────────

type preferenceManager struct {
	repo   repository.Repository
	logger *zerolog.Logger
}

// NewPreferenceManager creates a PreferenceManager backed by SQLite.
func NewPreferenceManager(repo repository.Repository, logger *zerolog.Logger) PreferenceManager {
	return &preferenceManager{
		repo:   repo,
		logger: logger,
	}
}

func (m *preferenceManager) Get(ctx context.Context, userID, key string) (*repository.UserPreferenceRow, error) {
	return m.repo.GetUserPreference(ctx, userID, key)
}

func (m *preferenceManager) GetAll(ctx context.Context, userID string) ([]repository.UserPreferenceRow, error) {
	return m.repo.ListUserPreferences(ctx, userID)
}

func (m *preferenceManager) Set(ctx context.Context, userID, key, value, source, confidence string) error {
	err := m.repo.UpsertUserPreference(ctx, repository.UpsertUserPreferenceParams{
		UserID:     userID,
		PrefKey:    key,
		PrefValue:  value,
		Source:     source,
		Confidence: confidence,
	})
	if err != nil {
		return fmt.Errorf("upsert preference %s: %w", key, err)
	}

	m.logger.Info().
		Str("user_id", userID).
		Str("key", key).
		Str("value", value).
		Str("source", source).
		Str("confidence", confidence).
		Msg("[pref] updated")

	return nil
}

func (m *preferenceManager) SetExplicit(ctx context.Context, userID, key, value string) error {
	return m.Set(ctx, userID, key, value, "explicit", "high")
}

func (m *preferenceManager) SetInferred(ctx context.Context, userID, key, value string) error {
	return m.Set(ctx, userID, key, value, "inferred", "medium")
}

func (m *preferenceManager) Delete(ctx context.Context, userID, key string) error {
	return m.repo.DeleteUserPreference(ctx, userID, key)
}

func (m *preferenceManager) LoadDefaults(ctx context.Context, userID string) error {
	defaults := map[string]string{
		domain.PrefKeyLanguage:       "id",
		domain.PrefKeyTone:           "casual",
		domain.PrefKeyVerbosity:      "normal",
		domain.PrefKeyResponseFormat: "plain",
	}

	for key, val := range defaults {
		existing, err := m.repo.GetUserPreference(ctx, userID, key)
		if err != nil {
			return fmt.Errorf("check default %s: %w", key, err)
		}
		if existing != nil {
			continue // don't override existing preference
		}
		if err := m.Set(ctx, userID, key, val, "default", "low"); err != nil {
			return err
		}
	}

	m.logger.Debug().
		Str("user_id", userID).
		Msg("[pref] defaults loaded")

	return nil
}

func (m *preferenceManager) ToStateOverrides(ctx context.Context, userID string) (map[string]string, error) {
	prefs, err := m.repo.ListUserPreferences(ctx, userID)
	if err != nil {
		return nil, err
	}

	result := make(map[string]string, len(prefs))
	for _, p := range prefs {
		result[p.PrefKey] = p.PrefValue
	}
	return result, nil
}
