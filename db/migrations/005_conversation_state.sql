-- Migration 005: Conversation state + user preferences for behavioral framework
-- All statements are idempotent (IF NOT EXISTS guards).

-- ── Per-user per-session conversation state ────────────────────────
-- Tracks ephemeral session-level state: language, tone, verbosity,
-- active task, temporary objectives, etc.
CREATE TABLE IF NOT EXISTS conversation_states (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    session_id TEXT NOT NULL DEFAULT 'default',
    state_json TEXT NOT NULL DEFAULT '{}',
    last_intent TEXT NOT NULL DEFAULT '',
    last_message_class TEXT NOT NULL DEFAULT '',
    active_task TEXT NOT NULL DEFAULT '',
    turn_count INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, session_id)
);

CREATE INDEX IF NOT EXISTS idx_cs_user_session ON conversation_states(user_id, session_id);

-- ── Persistent user preferences (survive across sessions) ──────────
-- Stores long-lived preferences: preferred language, display name,
-- response style, formatting prefs, coding prefs, etc.
CREATE TABLE IF NOT EXISTS user_preferences (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    pref_key TEXT NOT NULL,
    pref_value TEXT NOT NULL,
    source TEXT NOT NULL DEFAULT 'inferred',  -- 'explicit', 'inferred', 'default'
    confidence TEXT NOT NULL DEFAULT 'low',    -- 'high', 'medium', 'low'
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, pref_key)
);

CREATE INDEX IF NOT EXISTS idx_up_user ON user_preferences(user_id);
CREATE INDEX IF NOT EXISTS idx_up_key ON user_preferences(user_id, pref_key);
