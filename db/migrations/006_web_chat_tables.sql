-- Web Chat Sessions
CREATE TABLE IF NOT EXISTS web_chat_sessions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES dashboard_users(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    is_pinned BOOLEAN NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_web_chat_sessions_user ON web_chat_sessions(user_id);
CREATE INDEX idx_web_chat_sessions_updated ON web_chat_sessions(updated_at DESC);

-- Web Chat Messages
CREATE TABLE IF NOT EXISTS web_chat_messages (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL REFERENCES web_chat_sessions(id) ON DELETE CASCADE,
    role TEXT NOT NULL CHECK(role IN ('user', 'assistant', 'system')),
    content TEXT NOT NULL,
    model TEXT NOT NULL, -- local, openai, anthropic, dll
    tool_calls_json TEXT, -- menyimpan call tool kalau ada
    token_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_web_chat_messages_session ON web_chat_messages(session_id);

-- User API Keys (BYOK) - Encrypted at rest
CREATE TABLE IF NOT EXISTS user_api_keys (
    user_id TEXT NOT NULL REFERENCES dashboard_users(id) ON DELETE CASCADE,
    provider TEXT NOT NULL CHECK(provider IN ('openai', 'anthropic', 'gemini')),
    encrypted_key TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, provider)
);

-- File Upload Tracking for RAG
CREATE TABLE IF NOT EXISTS web_upload_jobs (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES dashboard_users(id) ON DELETE CASCADE,
    file_name TEXT NOT NULL,
    file_path TEXT NOT NULL,
    file_size INTEGER NOT NULL,
    mime_type TEXT NOT NULL,
    status TEXT NOT NULL CHECK(status IN ('pending', 'processing', 'completed', 'failed')),
    error_message TEXT,
    document_id TEXT, -- ID di Qdrant/documents jika RAG berhasil
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
