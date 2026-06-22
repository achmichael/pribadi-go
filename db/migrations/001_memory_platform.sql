-- Migration 001: Platform-agnostic identity, redesigned messages, FTS5, user_facts
-- All statements are idempotent (IF NOT EXISTS / IF EXISTS guards).

-- ── Platform-agnostic users ────────────────────────────────────────
CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,            -- UUID v4
    display_name TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- ── N:1 mapping: platform identity → internal user ─────────────────
CREATE TABLE IF NOT EXISTS platform_identities (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    platform TEXT NOT NULL,             -- 'whatsapp', 'telegram', …
    platform_user_id TEXT NOT NULL,     -- WA JID / TG chat_id / …
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(platform, platform_user_id)
);
CREATE INDEX IF NOT EXISTS idx_pi_user ON platform_identities(user_id);

-- ── Drop legacy messages table ───────────────────────────────────────
DROP TABLE IF EXISTS messages;

-- ── Redesigned messages with user_id, session, role ────────────────
CREATE TABLE IF NOT EXISTS messages_v2 (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    platform TEXT NOT NULL DEFAULT 'whatsapp',
    platform_msg_id TEXT NOT NULL DEFAULT '',
    session_id TEXT NOT NULL DEFAULT 'default',
    role TEXT NOT NULL DEFAULT 'user' CHECK(role IN ('user','assistant','system')),
    content TEXT NOT NULL,
    token_count INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_mv2_user_session
    ON messages_v2(user_id, session_id, created_at);
CREATE INDEX IF NOT EXISTS idx_mv2_platform_msg
    ON messages_v2(platform, platform_msg_id);

-- ── FTS5 full-text search (standard tokenizer) ────────────────────
CREATE VIRTUAL TABLE IF NOT EXISTS messages_fts USING fts5(
    content,
    content='messages_v2',
    content_rowid='id'
);

-- ── FTS5 trigram index (substring / CJK / mixed-lang search) ──────
CREATE VIRTUAL TABLE IF NOT EXISTS messages_fts_trigram USING fts5(
    content,
    content='messages_v2',
    content_rowid='id',
    tokenize='trigram'
);

-- ── Triggers: keep FTS in sync with messages_v2 ───────────────────
-- INSERT
CREATE TRIGGER IF NOT EXISTS messages_v2_ai AFTER INSERT ON messages_v2 BEGIN
    INSERT INTO messages_fts(rowid, content) VALUES (new.id, new.content);
    INSERT INTO messages_fts_trigram(rowid, content) VALUES (new.id, new.content);
END;

-- DELETE
CREATE TRIGGER IF NOT EXISTS messages_v2_ad AFTER DELETE ON messages_v2 BEGIN
    INSERT INTO messages_fts(messages_fts, rowid, content)
        VALUES ('delete', old.id, old.content);
    INSERT INTO messages_fts_trigram(messages_fts_trigram, rowid, content)
        VALUES ('delete', old.id, old.content);
END;

-- UPDATE
CREATE TRIGGER IF NOT EXISTS messages_v2_au AFTER UPDATE ON messages_v2 BEGIN
    INSERT INTO messages_fts(messages_fts, rowid, content)
        VALUES ('delete', old.id, old.content);
    INSERT INTO messages_fts_trigram(messages_fts_trigram, rowid, content)
        VALUES ('delete', old.id, old.content);
    INSERT INTO messages_fts(rowid, content) VALUES (new.id, new.content);
    INSERT INTO messages_fts_trigram(rowid, content) VALUES (new.id, new.content);
END;

-- ── Fact memory ───────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS user_facts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    fact_text TEXT NOT NULL,
    category TEXT NOT NULL DEFAULT 'general',
    source_message_id INTEGER REFERENCES messages_v2(id) ON DELETE SET NULL,
    is_active INTEGER NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_uf_user_active ON user_facts(user_id, is_active);
CREATE INDEX IF NOT EXISTS idx_uf_category ON user_facts(user_id, category);
