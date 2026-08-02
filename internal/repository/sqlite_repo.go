package repository

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/achmichael/pribadi-go/internal/repository/sqlc"
	_ "modernc.org/sqlite"
)

// ─── Repository interface (unchanged legacy + new methods) ─────────

// Repository defines the interface for database operations
type Repository interface {
	// ── Legacy SQLC methods (messages table is kept for whatsmeow compat) ──
	InsertMessage(ctx context.Context, arg sqlc.InsertMessageParams) (sqlc.Message, error)
	GetMessageByID(ctx context.Context, id int64) (sqlc.Message, error)
	GetMessageByWAID(ctx context.Context, waID string) (sqlc.Message, error)
	ListMessages(ctx context.Context, limit, offset int64) ([]sqlc.Message, error)
	ListMessagesByJID(ctx context.Context, jid string, limit, offset int64) ([]sqlc.Message, error)
	UpdateMessage(ctx context.Context, arg sqlc.UpdateMessageParams) (sqlc.Message, error)
	DeleteMessage(ctx context.Context, id int64) error
	DeleteMessagesByUserSession(ctx context.Context, userID, sessionID string) error

	GetDB() *sql.DB

	// Reminders
	InsertReminder(ctx context.Context, arg sqlc.InsertReminderParams) (sqlc.Reminder, error)
	GetReminderByID(ctx context.Context, id int64) (sqlc.Reminder, error)
	ListReminders(ctx context.Context, limit, offset int64) ([]sqlc.Reminder, error)
	ListRemindersByJID(ctx context.Context, jid string, limit, offset int64) ([]sqlc.Reminder, error)
	ListPendingReminders(ctx context.Context, scheduledAt time.Time) ([]sqlc.Reminder, error)
	UpdateReminder(ctx context.Context, arg sqlc.UpdateReminderParams) (sqlc.Reminder, error)
	MarkReminderAsSent(ctx context.Context, id int64) error
	DeleteReminder(ctx context.Context, id int64) error

	// Projects
	InsertProject(ctx context.Context, arg sqlc.InsertProjectParams) (sqlc.Project, error)
	GetProjectByID(ctx context.Context, id int64) (sqlc.Project, error)
	ListProjects(ctx context.Context, limit, offset int64) ([]sqlc.Project, error)
	ListProjectsByOwner(ctx context.Context, ownerJID string, limit, offset int64) ([]sqlc.Project, error)
	ListProjectsByStatus(ctx context.Context, status string, limit, offset int64) ([]sqlc.Project, error)
	UpdateProject(ctx context.Context, arg sqlc.UpdateProjectParams) (sqlc.Project, error)
	DeleteProject(ctx context.Context, id int64) error

	// WBS Tasks
	InsertWBSTask(ctx context.Context, arg sqlc.InsertWBSTaskParams) (sqlc.WbsTask, error)
	GetWBSTaskByID(ctx context.Context, id int64) (sqlc.WbsTask, error)
	ListWBSTasks(ctx context.Context, limit, offset int64) ([]sqlc.WbsTask, error)
	ListWBSTasksByProject(ctx context.Context, projectID, limit, offset int64) ([]sqlc.WbsTask, error)
	ListWBSTasksByStatus(ctx context.Context, status string, limit, offset int64) ([]sqlc.WbsTask, error)
	UpdateWBSTask(ctx context.Context, arg sqlc.UpdateWBSTaskParams) (sqlc.WbsTask, error)
	DeleteWBSTask(ctx context.Context, id int64) error

	// ── New: platform-agnostic messages ──
	InsertMessageV2(ctx context.Context, arg InsertMessageV2Params) (MessageV2, error)
	ListMessagesByUserSession(ctx context.Context, userID, sessionID string, limit int) ([]MessageV2, error)
	SearchMessagesFTS(ctx context.Context, userID, query string, limit int) ([]MessageV2, error)
	SearchMessagesTrigram(ctx context.Context, userID, query string, limit int) ([]MessageV2, error)

	// ── New: users & platform identities ──
	CreateUser(ctx context.Context, id, displayName string) error
	GetUserByID(ctx context.Context, id string) (*User, error)
	UpsertPlatformIdentity(ctx context.Context, platform, platformUserID, userID string) error
	GetUserIDByPlatform(ctx context.Context, platform, platformUserID string) (string, error)

	// ── New: user facts ──
	InsertFact(ctx context.Context, arg InsertFactParams) (int64, error)
	ListActiveFacts(ctx context.Context, userID string, limit int) ([]UserFact, error)
	DeactivateFact(ctx context.Context, factID int64) error
	UpdateFactTimestamp(ctx context.Context, factID int64) error

	// ── New: user documents ──
	InsertUserDocument(ctx context.Context, arg InsertUserDocumentParams) error
	GetLatestUserDocuments(ctx context.Context, userID string, limit int) ([]UserDocument, error)
	GetUserDocumentByID(ctx context.Context, id string) (*UserDocument, error)

	// ── New: conversation state ──
	GetConversationState(ctx context.Context, userID, sessionID string) (*ConversationStateRow, error)
	UpsertConversationState(ctx context.Context, arg UpsertConversationStateParams) error
	IncrementTurnCount(ctx context.Context, userID, sessionID string) error

	// ── New: user preferences ──
	GetUserPreference(ctx context.Context, userID, prefKey string) (*UserPreferenceRow, error)
	ListUserPreferences(ctx context.Context, userID string) ([]UserPreferenceRow, error)
	UpsertUserPreference(ctx context.Context, arg UpsertUserPreferenceParams) error
	DeleteUserPreference(ctx context.Context, userID, prefKey string) error

	// ── New: session state ──
	DeleteConversationState(ctx context.Context, userID, sessionID string) error

	Close() error
}

// ─── New model types ───────────────────────────────────────────────

type User struct {
	ID          string
	DisplayName string
	CreatedAt   time.Time
}

type MessageV2 struct {
	ID            int64
	UserID        string
	Platform      string
	PlatformMsgID string
	SessionID     string
	Role          string
	Content       string
	TokenCount    int
	CreatedAt     time.Time
}

type InsertMessageV2Params struct {
	UserID        string
	Platform      string
	PlatformMsgID string
	SessionID     string
	Role          string
	Content       string
	TokenCount    int
}

type UserFact struct {
	ID              int64
	UserID          string
	FactText        string
	Category        string
	SourceMessageID sql.NullInt64
	IsActive        int
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type InsertFactParams struct {
	UserID          string
	FactText        string
	Category        string
	SourceMessageID sql.NullInt64
}

type UserDocument struct {
	ID            string
	UserID        string
	PlatformMsgID string
	FileName      string
	Title         string
	Author        string
	MetadataJSON  string
	CreatedAt     time.Time
}

type InsertUserDocumentParams struct {
	ID            string
	UserID        string
	PlatformMsgID string
	FileName      string
	Title         string
	Author        string
	MetadataJSON  string
}

// ── Conversation State types ──

type ConversationStateRow struct {
	ID               int64
	UserID           string
	SessionID        string
	StateJSON        string
	LastIntent       string
	LastMessageClass string
	ActiveTask       string
	TurnCount        int
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type UpsertConversationStateParams struct {
	UserID           string
	SessionID        string
	StateJSON        string
	LastIntent       string
	LastMessageClass string
	ActiveTask       string
}

// ── User Preference types ──

type UserPreferenceRow struct {
	ID         int64
	UserID     string
	PrefKey    string
	PrefValue  string
	Source     string
	Confidence string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type UpsertUserPreferenceParams struct {
	UserID     string
	PrefKey    string
	PrefValue  string
	Source     string // "explicit", "inferred", "default"
	Confidence string // "high", "medium", "low"
}

// ─── Implementation ────────────────────────────────────────────────

type sqliteRepo struct {
	db      *sql.DB
	queries *sqlc.Queries
}

// NewSQLiteRepository creates a new SQLite repository with schema migration.
// schemaPath is relative to the working directory (e.g. "db/schema.sql").
// Migrations are loaded from a "migrations" sibling directory of schemaPath
// (e.g. "db/migrations/").
func NewSQLiteRepository(dbPath, schemaPath string) (Repository, error) {
	// Create directory if not exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create db directory: %w", err)
	}

	// Open database
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// ── WAL mode with NFS detection ──
	journalMode := "WAL"
	if isNetworkFS(dbPath) {
		journalMode = "DELETE"
		fmt.Fprintf(os.Stderr, "[WARN] SQLite DB on network filesystem (%s), falling back to DELETE journal mode\n", dbPath)
	}
	var resultMode string
	err = db.QueryRow(fmt.Sprintf("PRAGMA journal_mode=%s;", journalMode)).Scan(&resultMode)
	if err != nil {
		return nil, fmt.Errorf("failed to set journal_mode=%s: %w", journalMode, err)
	}
	if !strings.EqualFold(resultMode, journalMode) {
		fmt.Fprintf(os.Stderr, "[WARN] Requested journal_mode=%s but got %s\n", journalMode, resultMode)
	}

	// Enable foreign key support
	if _, err := db.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Run base schema (legacy tables)
	schemaSQL, err := os.ReadFile(schemaPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read schema file: %w", err)
	}
	if _, err := db.Exec(string(schemaSQL)); err != nil {
		return nil, fmt.Errorf("failed to execute schema: %w", err)
	}

	// Run migrations from sibling "migrations/" dir of schemaPath
	migrationsDir := filepath.Join(filepath.Dir(schemaPath), "migrations")
	if err := runMigrations(db, migrationsDir); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return &sqliteRepo{
		db:      db,
		queries: sqlc.New(db),
	}, nil
}

// isNetworkFS detects NFS/SMB/CIFS mounts via statfs.
func isNetworkFS(path string) bool {
	var stat syscall.Statfs_t
	dir := filepath.Dir(path)
	if err := syscall.Statfs(dir, &stat); err != nil {
		return false // can't detect → assume local
	}
	// NFS magic: 0x6969, SMB/CIFS: 0xFF534D42, SMB2: 0xFE534D42
	switch stat.Type {
	case 0x6969, 0xFF534D42, 0xFE534D42:
		return true
	}
	return false
}

// runMigrations executes all .sql files in migrationsDir sorted by name.
func runMigrations(db *sql.DB, migrationsDir string) error {
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // no migrations dir = nothing to do
		}
		return fmt.Errorf("read migrations dir: %w", err)
	}

	var sqlFiles []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			sqlFiles = append(sqlFiles, e.Name())
		}
	}
	sort.Strings(sqlFiles)

	for _, f := range sqlFiles {
		data, err := os.ReadFile(filepath.Join(migrationsDir, f))
		if err != nil {
			return fmt.Errorf("read migration %s: %w", f, err)
		}
		if _, err := db.Exec(string(data)); err != nil {
			// Ignore "duplicate column name" so ALTER TABLE ADD COLUMN is idempotent
			if strings.Contains(err.Error(), "duplicate column name") {
				continue
			}
			return fmt.Errorf("execute migration %s: %w", f, err)
		}
	}
	return nil
}

func (r *sqliteRepo) GetDB() *sql.DB {
	return r.db
}

func (r *sqliteRepo) Close() error {
	return r.db.Close()
}

// ═══════════════════════════════════════════════════════════════════
// Legacy SQLC delegations (unchanged)
// ═══════════════════════════════════════════════════════════════════

func (r *sqliteRepo) InsertMessage(ctx context.Context, arg sqlc.InsertMessageParams) (sqlc.Message, error) {
	return r.queries.InsertMessage(ctx, arg)
}
func (r *sqliteRepo) GetMessageByID(ctx context.Context, id int64) (sqlc.Message, error) {
	return r.queries.GetMessageByID(ctx, id)
}
func (r *sqliteRepo) GetMessageByWAID(ctx context.Context, waID string) (sqlc.Message, error) {
	return r.queries.GetMessageByWAID(ctx, waID)
}
func (r *sqliteRepo) ListMessages(ctx context.Context, limit, offset int64) ([]sqlc.Message, error) {
	return r.queries.ListMessages(ctx, sqlc.ListMessagesParams{Limit: limit, Offset: offset})
}
func (r *sqliteRepo) ListMessagesByJID(ctx context.Context, jid string, limit, offset int64) ([]sqlc.Message, error) {
	return r.queries.ListMessagesByJID(ctx, sqlc.ListMessagesByJIDParams{FromJid: jid, ToJid: jid, Limit: limit, Offset: offset})
}
func (r *sqliteRepo) UpdateMessage(ctx context.Context, arg sqlc.UpdateMessageParams) (sqlc.Message, error) {
	return r.queries.UpdateMessage(ctx, arg)
}
func (r *sqliteRepo) DeleteMessage(ctx context.Context, id int64) error {
	return r.queries.DeleteMessage(ctx, id)
}

// Reminders
func (r *sqliteRepo) InsertReminder(ctx context.Context, arg sqlc.InsertReminderParams) (sqlc.Reminder, error) {
	return r.queries.InsertReminder(ctx, arg)
}
func (r *sqliteRepo) GetReminderByID(ctx context.Context, id int64) (sqlc.Reminder, error) {
	return r.queries.GetReminderByID(ctx, id)
}
func (r *sqliteRepo) ListReminders(ctx context.Context, limit, offset int64) ([]sqlc.Reminder, error) {
	return r.queries.ListReminders(ctx, sqlc.ListRemindersParams{Limit: limit, Offset: offset})
}
func (r *sqliteRepo) ListRemindersByJID(ctx context.Context, jid string, limit, offset int64) ([]sqlc.Reminder, error) {
	return r.queries.ListRemindersByJID(ctx, sqlc.ListRemindersByJIDParams{Jid: jid, Limit: limit, Offset: offset})
}
func (r *sqliteRepo) ListPendingReminders(ctx context.Context, scheduledAt time.Time) ([]sqlc.Reminder, error) {
	return r.queries.ListPendingReminders(ctx, scheduledAt)
}
func (r *sqliteRepo) UpdateReminder(ctx context.Context, arg sqlc.UpdateReminderParams) (sqlc.Reminder, error) {
	return r.queries.UpdateReminder(ctx, arg)
}
func (r *sqliteRepo) MarkReminderAsSent(ctx context.Context, id int64) error {
	return r.queries.MarkReminderAsSent(ctx, id)
}
func (r *sqliteRepo) DeleteReminder(ctx context.Context, id int64) error {
	return r.queries.DeleteReminder(ctx, id)
}

// Projects
func (r *sqliteRepo) InsertProject(ctx context.Context, arg sqlc.InsertProjectParams) (sqlc.Project, error) {
	return r.queries.InsertProject(ctx, arg)
}
func (r *sqliteRepo) GetProjectByID(ctx context.Context, id int64) (sqlc.Project, error) {
	return r.queries.GetProjectByID(ctx, id)
}
func (r *sqliteRepo) ListProjects(ctx context.Context, limit, offset int64) ([]sqlc.Project, error) {
	return r.queries.ListProjects(ctx, sqlc.ListProjectsParams{Limit: limit, Offset: offset})
}
func (r *sqliteRepo) ListProjectsByOwner(ctx context.Context, ownerJID string, limit, offset int64) ([]sqlc.Project, error) {
	return r.queries.ListProjectsByOwner(ctx, sqlc.ListProjectsByOwnerParams{OwnerJid: ownerJID, Limit: limit, Offset: offset})
}
func (r *sqliteRepo) ListProjectsByStatus(ctx context.Context, status string, limit, offset int64) ([]sqlc.Project, error) {
	return r.queries.ListProjectsByStatus(ctx, sqlc.ListProjectsByStatusParams{Status: status, Limit: limit, Offset: offset})
}
func (r *sqliteRepo) UpdateProject(ctx context.Context, arg sqlc.UpdateProjectParams) (sqlc.Project, error) {
	return r.queries.UpdateProject(ctx, arg)
}
func (r *sqliteRepo) DeleteProject(ctx context.Context, id int64) error {
	return r.queries.DeleteProject(ctx, id)
}

// WBS Tasks
func (r *sqliteRepo) InsertWBSTask(ctx context.Context, arg sqlc.InsertWBSTaskParams) (sqlc.WbsTask, error) {
	return r.queries.InsertWBSTask(ctx, arg)
}
func (r *sqliteRepo) GetWBSTaskByID(ctx context.Context, id int64) (sqlc.WbsTask, error) {
	return r.queries.GetWBSTaskByID(ctx, id)
}
func (r *sqliteRepo) ListWBSTasks(ctx context.Context, limit, offset int64) ([]sqlc.WbsTask, error) {
	return r.queries.ListWBSTasks(ctx, sqlc.ListWBSTasksParams{Limit: limit, Offset: offset})
}
func (r *sqliteRepo) ListWBSTasksByProject(ctx context.Context, projectID, limit, offset int64) ([]sqlc.WbsTask, error) {
	return r.queries.ListWBSTasksByProject(ctx, sqlc.ListWBSTasksByProjectParams{ProjectID: projectID, Limit: limit, Offset: offset})
}
func (r *sqliteRepo) ListWBSTasksByStatus(ctx context.Context, status string, limit, offset int64) ([]sqlc.WbsTask, error) {
	return r.queries.ListWBSTasksByStatus(ctx, sqlc.ListWBSTasksByStatusParams{Status: status, Limit: limit, Offset: offset})
}
func (r *sqliteRepo) UpdateWBSTask(ctx context.Context, arg sqlc.UpdateWBSTaskParams) (sqlc.WbsTask, error) {
	return r.queries.UpdateWBSTask(ctx, arg)
}
func (r *sqliteRepo) DeleteWBSTask(ctx context.Context, id int64) error {
	return r.queries.DeleteWBSTask(ctx, id)
}

// ═══════════════════════════════════════════════════════════════════
// New: Users & Platform Identities
// ═══════════════════════════════════════════════════════════════════

func (r *sqliteRepo) CreateUser(ctx context.Context, id, displayName string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO users (id, display_name) VALUES (?, ?)`,
		id, displayName,
	)
	return err
}

func (r *sqliteRepo) GetUserByID(ctx context.Context, id string) (*User, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, display_name, created_at FROM users WHERE id = ?`, id,
	)
	var u User
	if err := row.Scan(&u.ID, &u.DisplayName, &u.CreatedAt); err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *sqliteRepo) UpsertPlatformIdentity(ctx context.Context, platform, platformUserID, userID string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO platform_identities (user_id, platform, platform_user_id)
		 VALUES (?, ?, ?)
		 ON CONFLICT(platform, platform_user_id) DO UPDATE SET user_id = excluded.user_id`,
		userID, platform, platformUserID,
	)
	return err
}

func (r *sqliteRepo) GetUserIDByPlatform(ctx context.Context, platform, platformUserID string) (string, error) {
	var userID string
	err := r.db.QueryRowContext(ctx,
		`SELECT user_id FROM platform_identities WHERE platform = ? AND platform_user_id = ?`,
		platform, platformUserID,
	).Scan(&userID)
	return userID, err
}

// ═══════════════════════════════════════════════════════════════════
// New: Messages V2 (platform-agnostic, with session & role)
// ═══════════════════════════════════════════════════════════════════

func (r *sqliteRepo) InsertMessageV2(ctx context.Context, arg InsertMessageV2Params) (MessageV2, error) {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO messages_v2 (user_id, platform, platform_msg_id, session_id, role, content, token_count)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		arg.UserID, arg.Platform, arg.PlatformMsgID, arg.SessionID, arg.Role, arg.Content, arg.TokenCount,
	)
	if err != nil {
		return MessageV2{}, err
	}
	id, _ := res.LastInsertId()

	return MessageV2{
		ID:            id,
		UserID:        arg.UserID,
		Platform:      arg.Platform,
		PlatformMsgID: arg.PlatformMsgID,
		SessionID:     arg.SessionID,
		Role:          arg.Role,
		Content:       arg.Content,
		TokenCount:    arg.TokenCount,
	}, nil
}

func (r *sqliteRepo) ListMessagesByUserSession(ctx context.Context, userID, sessionID string, limit int) ([]MessageV2, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, user_id, platform, platform_msg_id, session_id, role, content, token_count, created_at
		 FROM (
			 SELECT id, user_id, platform, platform_msg_id, session_id, role, content, token_count, created_at
			 FROM messages_v2
			 WHERE user_id = ? AND session_id = ?
			 ORDER BY created_at DESC
			 LIMIT ?
		 ) sub
		 ORDER BY created_at ASC`,
		userID, sessionID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMessagesV2(rows)
}

// SearchMessagesFTS performs full-text search using FTS5 standard tokenizer.
// Query follows FTS5 query syntax (e.g. "golang AND project").
func (r *sqliteRepo) SearchMessagesFTS(ctx context.Context, userID, query string, limit int) ([]MessageV2, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT m.id, m.user_id, m.platform, m.platform_msg_id, m.session_id, m.role, m.content, m.token_count, m.created_at
		 FROM messages_fts f
		 JOIN messages_v2 m ON m.id = f.rowid
		 WHERE f.messages_fts MATCH ?
		   AND m.user_id = ?
		 ORDER BY rank
		 LIMIT ?`,
		query, userID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMessagesV2(rows)
}

// SearchMessagesTrigram performs substring search using FTS5 trigram tokenizer.
// Good for partial matches, CJK, and mixed Indonesian-English terms.
func (r *sqliteRepo) SearchMessagesTrigram(ctx context.Context, userID, query string, limit int) ([]MessageV2, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT m.id, m.user_id, m.platform, m.platform_msg_id, m.session_id, m.role, m.content, m.token_count, m.created_at
		 FROM messages_fts_trigram f
		 JOIN messages_v2 m ON m.id = f.rowid
		 WHERE f.messages_fts_trigram MATCH ?
		   AND m.user_id = ?
		 ORDER BY rank
		 LIMIT ?`,
		query, userID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMessagesV2(rows)
}

func scanMessagesV2(rows *sql.Rows) ([]MessageV2, error) {
	var results []MessageV2
	for rows.Next() {
		var m MessageV2
		if err := rows.Scan(
			&m.ID, &m.UserID, &m.Platform, &m.PlatformMsgID,
			&m.SessionID, &m.Role, &m.Content, &m.TokenCount, &m.CreatedAt,
		); err != nil {
			return nil, err
		}
		results = append(results, m)
	}
	return results, rows.Err()
}

// ═══════════════════════════════════════════════════════════════════
// New: User Facts
// ═══════════════════════════════════════════════════════════════════

func (r *sqliteRepo) InsertFact(ctx context.Context, arg InsertFactParams) (int64, error) {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO user_facts (user_id, fact_text, category, source_message_id)
		 VALUES (?, ?, ?, ?)`,
		arg.UserID, arg.FactText, arg.Category, arg.SourceMessageID,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r *sqliteRepo) ListActiveFacts(ctx context.Context, userID string, limit int) ([]UserFact, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, user_id, fact_text, category, source_message_id, is_active, created_at, updated_at
		 FROM user_facts
		 WHERE user_id = ? AND is_active = 1
		 ORDER BY updated_at DESC
		 LIMIT ?`,
		userID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []UserFact
	for rows.Next() {
		var f UserFact
		if err := rows.Scan(
			&f.ID, &f.UserID, &f.FactText, &f.Category,
			&f.SourceMessageID, &f.IsActive, &f.CreatedAt, &f.UpdatedAt,
		); err != nil {
			return nil, err
		}
		results = append(results, f)
	}
	return results, rows.Err()
}

func (r *sqliteRepo) DeactivateFact(ctx context.Context, factID int64) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE user_facts SET is_active = 0, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		factID,
	)
	return err
}

func (r *sqliteRepo) UpdateFactTimestamp(ctx context.Context, factID int64) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE user_facts SET updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		factID,
	)
	return err
}

// ═══════════════════════════════════════════════════════════════════
// New: User Documents
// ═══════════════════════════════════════════════════════════════════

func (r *sqliteRepo) InsertUserDocument(ctx context.Context, arg InsertUserDocumentParams) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO user_documents (id, user_id, platform_msg_id, file_name, title, author, metadata_json)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		arg.ID, arg.UserID, arg.PlatformMsgID, arg.FileName, arg.Title, arg.Author, arg.MetadataJSON,
	)
	return err
}

func (r *sqliteRepo) GetLatestUserDocuments(ctx context.Context, userID string, limit int) ([]UserDocument, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, user_id, platform_msg_id, file_name, title, author, metadata_json, created_at
		 FROM user_documents
		 WHERE user_id = ?
		 ORDER BY created_at DESC
		 LIMIT ?`,
		userID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []UserDocument
	for rows.Next() {
		var d UserDocument
		if err := rows.Scan(
			&d.ID, &d.UserID, &d.PlatformMsgID,
			&d.FileName, &d.Title, &d.Author, &d.MetadataJSON, &d.CreatedAt,
		); err != nil {
			return nil, err
		}
		results = append(results, d)
	}
	return results, rows.Err()
}

func (r *sqliteRepo) GetUserDocumentByID(ctx context.Context, id string) (*UserDocument, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, user_id, platform_msg_id, file_name, title, author, metadata_json, created_at
		 FROM user_documents
		 WHERE id = ?`,
		id,
	)

	var d UserDocument
	if err := row.Scan(
		&d.ID, &d.UserID, &d.PlatformMsgID,
		&d.FileName, &d.Title, &d.Author, &d.MetadataJSON, &d.CreatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &d, nil
}

// ═════════════════════════════════════════════════════════════════
// New: Conversation State
// ═════════════════════════════════════════════════════════════════

func (r *sqliteRepo) GetConversationState(ctx context.Context, userID, sessionID string) (*ConversationStateRow, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, user_id, session_id, state_json, last_intent, last_message_class, active_task, turn_count, created_at, updated_at
		 FROM conversation_states
		 WHERE user_id = ? AND session_id = ?`,
		userID, sessionID,
	)
	var s ConversationStateRow
	if err := row.Scan(
		&s.ID, &s.UserID, &s.SessionID, &s.StateJSON,
		&s.LastIntent, &s.LastMessageClass, &s.ActiveTask,
		&s.TurnCount, &s.CreatedAt, &s.UpdatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

func (r *sqliteRepo) UpsertConversationState(ctx context.Context, arg UpsertConversationStateParams) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO conversation_states (user_id, session_id, state_json, last_intent, last_message_class, active_task, turn_count)
		 VALUES (?, ?, ?, ?, ?, ?, 1)
		 ON CONFLICT(user_id, session_id) DO UPDATE SET
			state_json = excluded.state_json,
			last_intent = excluded.last_intent,
			last_message_class = excluded.last_message_class,
			active_task = excluded.active_task,
			turn_count = conversation_states.turn_count + 1,
			updated_at = CURRENT_TIMESTAMP`,
		arg.UserID, arg.SessionID, arg.StateJSON, arg.LastIntent, arg.LastMessageClass, arg.ActiveTask,
	)
	return err
}

func (r *sqliteRepo) IncrementTurnCount(ctx context.Context, userID, sessionID string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE conversation_states SET turn_count = turn_count + 1, updated_at = CURRENT_TIMESTAMP
		 WHERE user_id = ? AND session_id = ?`,
		userID, sessionID,
	)
	return err
}

// ═════════════════════════════════════════════════════════════════
// New: User Preferences
// ═════════════════════════════════════════════════════════════════

func (r *sqliteRepo) GetUserPreference(ctx context.Context, userID, prefKey string) (*UserPreferenceRow, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, user_id, pref_key, pref_value, source, confidence, created_at, updated_at
		 FROM user_preferences
		 WHERE user_id = ? AND pref_key = ?`,
		userID, prefKey,
	)
	var p UserPreferenceRow
	if err := row.Scan(
		&p.ID, &p.UserID, &p.PrefKey, &p.PrefValue,
		&p.Source, &p.Confidence, &p.CreatedAt, &p.UpdatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

func (r *sqliteRepo) ListUserPreferences(ctx context.Context, userID string) ([]UserPreferenceRow, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, user_id, pref_key, pref_value, source, confidence, created_at, updated_at
		 FROM user_preferences
		 WHERE user_id = ?
		 ORDER BY pref_key`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []UserPreferenceRow
	for rows.Next() {
		var p UserPreferenceRow
		if err := rows.Scan(
			&p.ID, &p.UserID, &p.PrefKey, &p.PrefValue,
			&p.Source, &p.Confidence, &p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, err
		}
		results = append(results, p)
	}
	return results, rows.Err()
}

func (r *sqliteRepo) UpsertUserPreference(ctx context.Context, arg UpsertUserPreferenceParams) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO user_preferences (user_id, pref_key, pref_value, source, confidence)
		 VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT(user_id, pref_key) DO UPDATE SET
			pref_value = excluded.pref_value,
			source = excluded.source,
			confidence = excluded.confidence,
			updated_at = CURRENT_TIMESTAMP`,
		arg.UserID, arg.PrefKey, arg.PrefValue, arg.Source, arg.Confidence,
	)
	return err
}

func (r *sqliteRepo) DeleteUserPreference(ctx context.Context, userID, prefKey string) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM user_preferences WHERE user_id = ? AND pref_key = ?`,
		userID, prefKey,
	)
	return err
}

func (r *sqliteRepo) DeleteMessagesByUserSession(ctx context.Context, userID, sessionID string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM messages_v2 WHERE user_id = ? AND session_id = ?", userID, sessionID)
	if err != nil {
		return err
	}
	return nil
}

func (r *sqliteRepo) DeleteConversationState(ctx context.Context, userID, sessionID string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM conversation_states WHERE user_id = ? AND session_id = ?", userID, sessionID)
	if err != nil {
		return err
	}
	return nil
}
