-- Migration 009: Add metadata column to messages_v2 for storing truncation info and other metadata
-- This column stores JSON data including: truncated (bool), done_reason (string), etc.

ALTER TABLE messages_v2 ADD COLUMN metadata TEXT DEFAULT '{}';

CREATE INDEX IF NOT EXISTS idx_mv2_metadata ON messages_v2(user_id, session_id, metadata);
