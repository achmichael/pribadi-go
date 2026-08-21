package domain

import "time"

// WebChatSession represents a chat session in the dashboard.
type WebChatSession struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Title     string    `json:"title"`
	IsPinned  bool      `json:"is_pinned"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// WebChatMessage represents a message within a web chat session.
type WebChatMessage struct {
	ID            string    `json:"id"`
	SessionID     string    `json:"session_id"`
	Role          string    `json:"role"`
	Content       string    `json:"content"`
	Model         string    `json:"model"`
	ToolCallsJSON string    `json:"tool_calls_json,omitempty"`
	TokenCount    int       `json:"token_count"`
	CreatedAt     time.Time `json:"created_at"`
}

// UserAPIKey represents an encrypted BYOK API key.
type UserAPIKey struct {
	UserID       string    `json:"user_id"`
	Provider     string    `json:"provider"`
	EncryptedKey string    `json:"encrypted_key"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// WebUploadJob tracks a file uploaded for RAG ingestion.
type WebUploadJob struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	FileName     string    `json:"file_name"`
	FilePath     string    `json:"file_path"`
	FileSize     int64     `json:"file_size"`
	MimeType     string    `json:"mime_type"`
	Status       string    `json:"status"` // pending, processing, completed, failed
	ErrorMessage string    `json:"error_message,omitempty"`
	DocumentID   string    `json:"document_id,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
