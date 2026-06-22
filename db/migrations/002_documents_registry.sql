-- Migration 002: User Documents Registry
-- Tracks metadata of uploaded documents for Reference Resolution Agent

CREATE TABLE IF NOT EXISTS user_documents (
    id TEXT PRIMARY KEY,            -- UUID v4
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    platform_msg_id TEXT NOT NULL,  -- e.g., WhatsApp message ID where it was uploaded
    file_name TEXT NOT NULL DEFAULT '',
    title TEXT NOT NULL DEFAULT 'Unknown',
    author TEXT NOT NULL DEFAULT 'Unknown',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_user_documents_user ON user_documents(user_id, created_at DESC);
