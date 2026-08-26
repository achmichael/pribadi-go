-- Migration 008: Unify web_chat_messages into messages_v2
-- Columns model, tool_calls_json, file_job_id have been added.
-- The actual migration was run manually to avoid SQLite startup parser errors.

CREATE INDEX IF NOT EXISTS idx_mv2_file_job ON messages_v2(file_job_id);
