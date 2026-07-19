package domain

import "time"

// ConversationState tracks ephemeral per-user per-session state.
// state_json holds a serialized StateData struct.
type ConversationState struct {
	ID               int64     `json:"id"`
	UserID           string    `json:"user_id"`
	SessionID        string    `json:"session_id"`
	StateJSON        string    `json:"state_json"`
	LastIntent       string    `json:"last_intent"`
	LastMessageClass string    `json:"last_message_class"`
	ActiveTask       string    `json:"active_task"`
	TurnCount        int       `json:"turn_count"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// StateData is the structured content stored in state_json.
// Serialized/deserialized by StateManager.
type StateData struct {
	Language          string            `json:"language"`           // e.g. "id", "en"
	Tone              string            `json:"tone"`               // "formal", "casual", "friendly"
	Verbosity         string            `json:"verbosity"`          // "concise", "normal", "detailed"
	ResponseFormat    string            `json:"response_format"`    // "plain", "markdown", "step_by_step"
	UserDisplayName   string            `json:"user_display_name"`  // "Call me Michael"
	ActiveDocumentID  string            `json:"active_document_id"` // currently focused document
	PendingClarify    string            `json:"pending_clarify"`    // awaiting clarification on what
	LastCorrectionAt  string            `json:"last_correction_at"` // ISO timestamp of last correction
	CustomDirectives  map[string]string `json:"custom_directives"`  // arbitrary user instructions
}

// DefaultStateData returns sensible defaults.
func DefaultStateData() StateData {
	return StateData{
		Language:         "id",
		Tone:             "casual",
		Verbosity:        "normal",
		ResponseFormat:   "plain",
		CustomDirectives: make(map[string]string),
	}
}

// UserPreference is a persistent key-value preference for a user.
type UserPreference struct {
	ID        int64     `json:"id"`
	UserID    string    `json:"user_id"`
	PrefKey   string    `json:"pref_key"`
	PrefValue string    `json:"pref_value"`
	Source    string    `json:"source"`     // "explicit", "inferred", "default"
	Confidence string  `json:"confidence"` // "high", "medium", "low"
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Well-known preference keys.
const (
	PrefKeyLanguage       = "language"
	PrefKeyTone           = "tone"
	PrefKeyVerbosity      = "verbosity"
	PrefKeyResponseFormat = "response_format"
	PrefKeyDisplayName    = "display_name"
)
