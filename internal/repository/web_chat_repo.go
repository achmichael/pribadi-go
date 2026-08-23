package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/achmichael/pribadi-go/internal/domain"
)

// WebChatRepository provides data access for web chat functionality
type WebChatRepository interface {
	// Sessions
	CreateSession(ctx context.Context, session domain.WebChatSession) error
	GetSession(ctx context.Context, sessionID string) (*domain.WebChatSession, error)
	ListSessions(ctx context.Context, userID string) ([]domain.WebChatSession, error)
	UpdateSession(ctx context.Context, session domain.WebChatSession) error
	DeleteSession(ctx context.Context, sessionID, userID string) error

	// Messages
	CreateMessage(ctx context.Context, msg domain.WebChatMessage) error
	ListMessages(ctx context.Context, sessionID string) ([]domain.WebChatMessage, error)

	// API Keys
	UpsertAPIKey(ctx context.Context, key domain.UserAPIKey) error
	GetAPIKey(ctx context.Context, userID, provider string) (*domain.UserAPIKey, error)
	DeleteAPIKey(ctx context.Context, userID, provider string) error

	// Upload Jobs
	CreateUploadJob(ctx context.Context, job domain.WebUploadJob) error
	GetUploadJob(ctx context.Context, jobID string) (*domain.WebUploadJob, error)
	UpdateUploadJobStatus(ctx context.Context, jobID, status, errMsg, docID string) error
}

type webChatRepo struct {
	db *sql.DB
}

func NewWebChatRepository(db *sql.DB) WebChatRepository {
	return &webChatRepo{db: db}
}

func (r *webChatRepo) CreateSession(ctx context.Context, s domain.WebChatSession) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO web_chat_sessions (id, user_id, title, is_pinned, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		s.ID, s.UserID, s.Title, s.IsPinned, s.CreatedAt, s.UpdatedAt,
	)
	return err
}

func (r *webChatRepo) GetSession(ctx context.Context, sessionID string) (*domain.WebChatSession, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, user_id, title, is_pinned, created_at, updated_at FROM web_chat_sessions WHERE id = ?`, sessionID)
	var s domain.WebChatSession
	err := row.Scan(&s.ID, &s.UserID, &s.Title, &s.IsPinned, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

func (r *webChatRepo) ListSessions(ctx context.Context, userID string) ([]domain.WebChatSession, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, user_id, title, is_pinned, created_at, updated_at FROM web_chat_sessions WHERE user_id = ? ORDER BY is_pinned DESC, updated_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []domain.WebChatSession
	for rows.Next() {
		var s domain.WebChatSession
		if err := rows.Scan(&s.ID, &s.UserID, &s.Title, &s.IsPinned, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		res = append(res, s)
	}
	return res, nil
}

func (r *webChatRepo) UpdateSession(ctx context.Context, s domain.WebChatSession) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE web_chat_sessions SET title = ?, is_pinned = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ? AND user_id = ?`,
		s.Title, s.IsPinned, s.ID, s.UserID,
	)
	return err
}

func (r *webChatRepo) DeleteSession(ctx context.Context, sessionID, userID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM web_chat_sessions WHERE id = ? AND user_id = ?`, sessionID, userID)
	return err
}

func (r *webChatRepo) CreateMessage(ctx context.Context, m domain.WebChatMessage) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO web_chat_messages (id, session_id, role, content, model, tool_calls_json, token_count, created_at, file_job_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		m.ID, m.SessionID, m.Role, m.Content, m.Model, m.ToolCallsJSON, m.TokenCount, m.CreatedAt, m.FileJobID,
	)
	if err == nil {
		r.db.ExecContext(ctx, `UPDATE web_chat_sessions SET updated_at = CURRENT_TIMESTAMP WHERE id = ?`, m.SessionID)
	}
	return err
}

func (r *webChatRepo) ListMessages(ctx context.Context, sessionID string) ([]domain.WebChatMessage, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT 
			m.id, m.session_id, m.role, m.content, m.model, m.tool_calls_json, m.token_count, m.created_at, m.file_job_id,
			j.file_name, j.mime_type, j.file_size
		FROM web_chat_messages m
		LEFT JOIN web_upload_jobs j ON m.file_job_id = j.id
		WHERE m.session_id = ? 
		ORDER BY m.created_at ASC`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []domain.WebChatMessage
	for rows.Next() {
		var m domain.WebChatMessage
		var tc sql.NullString
		var fileJobID sql.NullString
		var fileName sql.NullString
		var mimeType sql.NullString
		var fileSize sql.NullInt64

		if err := rows.Scan(
			&m.ID, &m.SessionID, &m.Role, &m.Content, &m.Model, &tc, &m.TokenCount, &m.CreatedAt, &fileJobID,
			&fileName, &mimeType, &fileSize,
		); err != nil {
			return nil, err
		}
		
		if tc.Valid {
			m.ToolCallsJSON = tc.String
		}
		if fileJobID.Valid {
			m.FileJobID = fileJobID.String
		}
		if fileName.Valid {
			m.FileName = fileName.String
		}
		if mimeType.Valid {
			m.FileMimeType = mimeType.String
		}
		if fileSize.Valid {
			m.FileSize = fileSize.Int64
		}
		
		res = append(res, m)
	}
	return res, nil
}

func (r *webChatRepo) UpsertAPIKey(ctx context.Context, k domain.UserAPIKey) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO user_api_keys (user_id, provider, encrypted_key) VALUES (?, ?, ?)
		 ON CONFLICT(user_id, provider) DO UPDATE SET encrypted_key = excluded.encrypted_key, updated_at = CURRENT_TIMESTAMP`,
		k.UserID, k.Provider, k.EncryptedKey,
	)
	return err
}

func (r *webChatRepo) GetAPIKey(ctx context.Context, userID, provider string) (*domain.UserAPIKey, error) {
	row := r.db.QueryRowContext(ctx, `SELECT user_id, provider, encrypted_key, created_at, updated_at FROM user_api_keys WHERE user_id = ? AND provider = ?`, userID, provider)
	var k domain.UserAPIKey
	err := row.Scan(&k.UserID, &k.Provider, &k.EncryptedKey, &k.CreatedAt, &k.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &k, nil
}

func (r *webChatRepo) DeleteAPIKey(ctx context.Context, userID, provider string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM user_api_keys WHERE user_id = ? AND provider = ?`, userID, provider)
	return err
}

func (r *webChatRepo) CreateUploadJob(ctx context.Context, j domain.WebUploadJob) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO web_upload_jobs (id, user_id, file_name, file_path, file_size, mime_type, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		j.ID, j.UserID, j.FileName, j.FilePath, j.FileSize, j.MimeType, j.Status, j.CreatedAt, j.UpdatedAt,
	)
	return err
}

func (r *webChatRepo) GetUploadJob(ctx context.Context, jobID string) (*domain.WebUploadJob, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, user_id, file_name, file_path, file_size, mime_type, status, error_message, document_id, created_at, updated_at FROM web_upload_jobs WHERE id = ?`, jobID)
	var j domain.WebUploadJob
	var errMsg, docID sql.NullString
	err := row.Scan(&j.ID, &j.UserID, &j.FileName, &j.FilePath, &j.FileSize, &j.MimeType, &j.Status, &errMsg, &docID, &j.CreatedAt, &j.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if errMsg.Valid {
		j.ErrorMessage = errMsg.String
	}
	if docID.Valid {
		j.DocumentID = docID.String
	}
	return &j, nil
}

func (r *webChatRepo) UpdateUploadJobStatus(ctx context.Context, jobID, status, errMsg, docID string) error {
	var errStr, docStr sql.NullString
	if errMsg != "" {
		errStr = sql.NullString{String: errMsg, Valid: true}
	}
	if docID != "" {
		docStr = sql.NullString{String: docID, Valid: true}
	}
	_, err := r.db.ExecContext(ctx,
		`UPDATE web_upload_jobs SET status = ?, error_message = ?, document_id = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		status, errStr, docStr, jobID,
	)
	return err
}
