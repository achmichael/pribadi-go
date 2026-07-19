package classifier

import (
	"context"
	"fmt"
	"time"

	"github.com/achmichael/pribadi-go/internal/conversation"
	"github.com/achmichael/pribadi-go/internal/domain"
	"github.com/rs/zerolog"
)

// ─── Interface ─────────────────────────────────────────────────────

// Interceptor processes behavioral instructions and corrections.
// It updates conversation state and preferences without forwarding
// the message to the LLM for a full response (Principle 4, 8).
type Interceptor interface {
	// Process handles a classified message. Returns:
	// - intercepted=true if message was a pure instruction (no LLM response needed)
	// - ackMessage: short acknowledgement to send back (empty if not intercepted)
	// - error
	Process(ctx context.Context, params InterceptParams) (intercepted bool, ackMessage string, err error)
}

// InterceptParams holds inputs for instruction interception.
type InterceptParams struct {
	UserID    string
	SessionID string
	UserText  string
	Class     *Classification
	State     *domain.StateData
}

// ─── Implementation ────────────────────────────────────────────────

type interceptor struct {
	stateManager conversation.StateManager
	prefManager  conversation.PreferenceManager
	logger       *zerolog.Logger
}

// NewInterceptor creates an Interceptor.
func NewInterceptor(
	stateManager conversation.StateManager,
	prefManager conversation.PreferenceManager,
	logger *zerolog.Logger,
) Interceptor {
	return &interceptor{
		stateManager: stateManager,
		prefManager:  prefManager,
		logger:       logger,
	}
}

func (i *interceptor) Process(ctx context.Context, params InterceptParams) (bool, string, error) {
	start := time.Now()

	if params.Class == nil {
		return false, "", nil
	}

	// Not an instruction or correction → not intercepted
	if !params.Class.IsInstruction && !params.Class.IsCorrection {
		return false, "", nil
	}

	// Handle preference updates
	if params.Class.IsInstruction && len(params.Class.ExtractedPrefs) > 0 {
		ack, err := i.applyPreferences(ctx, params)
		if err != nil {
			return false, "", err
		}

		i.logger.Info().
			Str("user_id", params.UserID).
			Int("prefs_applied", len(params.Class.ExtractedPrefs)).
			Dur("ms", time.Since(start)).
			Msg("[interceptor] preferences applied")

		return true, ack, nil
	}

	// Handle correction
	if params.Class.IsCorrection {
		err := i.handleCorrection(ctx, params)
		if err != nil {
			return false, "", err
		}

		i.logger.Info().
			Str("user_id", params.UserID).
			Dur("ms", time.Since(start)).
			Msg("[interceptor] correction noted")

		// Corrections still need LLM response with corrected context
		// So we DON'T intercept — but we've updated the state
		return false, "", nil
	}

	return false, "", nil
}

// applyPreferences updates both session state and persistent preferences.
func (i *interceptor) applyPreferences(ctx context.Context, params InterceptParams) (string, error) {
	state := params.State
	if state == nil {
		defaultState := domain.DefaultStateData()
		state = &defaultState
	}

	var ackParts []string

	for _, pref := range params.Class.ExtractedPrefs {
		// Update persistent preference
		if err := i.prefManager.SetExplicit(ctx, params.UserID, pref.Key, pref.Value); err != nil {
			i.logger.Warn().Err(err).
				Str("key", pref.Key).
				Msg("[interceptor] failed to persist preference")
		}

		// Update session state
		switch pref.Key {
		case domain.PrefKeyLanguage:
			state.Language = pref.Value
			ackParts = append(ackParts, i.langAck(pref.Value))
		case domain.PrefKeyTone:
			state.Tone = pref.Value
			ackParts = append(ackParts, fmt.Sprintf("Tone → %s", pref.Value))
		case domain.PrefKeyVerbosity:
			state.Verbosity = pref.Value
			ackParts = append(ackParts, i.verbosityAck(pref.Value))
		case domain.PrefKeyResponseFormat:
			state.ResponseFormat = pref.Value
			ackParts = append(ackParts, fmt.Sprintf("Format → %s", pref.Value))
		case domain.PrefKeyDisplayName:
			state.UserDisplayName = pref.Value
			ackParts = append(ackParts, fmt.Sprintf("Baik, saya panggil kamu %s 👋", pref.Value))
		default:
			// Store as custom directive
			if state.CustomDirectives == nil {
				state.CustomDirectives = make(map[string]string)
			}
			state.CustomDirectives[pref.Key] = pref.Value
			ackParts = append(ackParts, fmt.Sprintf("%s → %s", pref.Key, pref.Value))
		}
	}

	// Persist updated state
	if err := i.stateManager.Update(ctx, params.UserID, params.SessionID, state,
		string(IntentSetPreference), string(ClassPreference), ""); err != nil {
		return "", fmt.Errorf("update state after preference: %w", err)
	}

	if len(ackParts) == 0 {
		return "✅", nil
	}
	if len(ackParts) == 1 {
		return "✅ " + ackParts[0], nil
	}
	return "✅ " + joinAcks(ackParts), nil
}

// handleCorrection marks correction in state.
func (i *interceptor) handleCorrection(ctx context.Context, params InterceptParams) error {
	state := params.State
	if state == nil {
		defaultState := domain.DefaultStateData()
		state = &defaultState
	}

	state.LastCorrectionAt = time.Now().UTC().Format(time.RFC3339)

	return i.stateManager.Update(ctx, params.UserID, params.SessionID, state,
		string(IntentCorrect), string(ClassCorrection), "")
}

func (i *interceptor) langAck(lang string) string {
	switch lang {
	case "en":
		return "Language → English"
	case "id":
		return "Bahasa → Indonesia"
	default:
		return fmt.Sprintf("Language → %s", lang)
	}
}

func (i *interceptor) verbosityAck(v string) string {
	switch v {
	case "concise":
		return "Mode → ringkas"
	case "detailed":
		return "Mode → detail"
	default:
		return fmt.Sprintf("Verbosity → %s", v)
	}
}

func joinAcks(parts []string) string {
	result := ""
	for idx, p := range parts {
		if idx > 0 {
			result += " | "
		}
		result += p
	}
	return result
}
