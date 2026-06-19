package repository

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/achmichael/pribadi-go/internal/repository/sqlc"
	_ "modernc.org/sqlite"
)

// Repository defines the interface for database operations
type Repository interface {
	// Messages
	InsertMessage(ctx context.Context, arg sqlc.InsertMessageParams) (sqlc.Message, error)
	GetMessageByID(ctx context.Context, id int64) (sqlc.Message, error)
	GetMessageByWAID(ctx context.Context, waID string) (sqlc.Message, error)
	ListMessages(ctx context.Context, limit, offset int64) ([]sqlc.Message, error)
	ListMessagesByJID(ctx context.Context, jid string, limit, offset int64) ([]sqlc.Message, error)
	UpdateMessage(ctx context.Context, arg sqlc.UpdateMessageParams) (sqlc.Message, error)
	DeleteMessage(ctx context.Context, id int64) error

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

	Close() error
}

type sqliteRepo struct {
	db      *sql.DB
	queries *sqlc.Queries
}

// NewSQLiteRepository creates a new SQLite repository with schema migration
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

	// Enable foreign key support for PostgreSQL compatibility
	if _, err := db.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Run schema migration
	schemaSQL, err := os.ReadFile(schemaPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read schema file: %w", err)
	}

	if _, err := db.Exec(string(schemaSQL)); err != nil {
		return nil, fmt.Errorf("failed to execute schema: %w", err)
	}

	return &sqliteRepo{
		db:      db,
		queries: sqlc.New(db),
	}, nil
}

func (r *sqliteRepo) GetDB() *sql.DB {
	return r.db
}

func (r *sqliteRepo) Close() error {
	return r.db.Close()
}

// Messages
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
	return r.queries.ListMessages(ctx, sqlc.ListMessagesParams{
		Limit:  limit,
		Offset: offset,
	})
}

func (r *sqliteRepo) ListMessagesByJID(ctx context.Context, jid string, limit, offset int64) ([]sqlc.Message, error) {
	return r.queries.ListMessagesByJID(ctx, sqlc.ListMessagesByJIDParams{
		FromJid: jid,
		ToJid:   jid,
		Limit:   limit,
		Offset:  offset,
	})
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
	return r.queries.ListReminders(ctx, sqlc.ListRemindersParams{
		Limit:  limit,
		Offset: offset,
	})
}

func (r *sqliteRepo) ListRemindersByJID(ctx context.Context, jid string, limit, offset int64) ([]sqlc.Reminder, error) {
	return r.queries.ListRemindersByJID(ctx, sqlc.ListRemindersByJIDParams{
		Jid:    jid,
		Limit:  limit,
		Offset: offset,
	})
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
	return r.queries.ListProjects(ctx, sqlc.ListProjectsParams{
		Limit:  limit,
		Offset: offset,
	})
}

func (r *sqliteRepo) ListProjectsByOwner(ctx context.Context, ownerJID string, limit, offset int64) ([]sqlc.Project, error) {
	return r.queries.ListProjectsByOwner(ctx, sqlc.ListProjectsByOwnerParams{
		OwnerJid: ownerJID,
		Limit:    limit,
		Offset:   offset,
	})
}

func (r *sqliteRepo) ListProjectsByStatus(ctx context.Context, status string, limit, offset int64) ([]sqlc.Project, error) {
	return r.queries.ListProjectsByStatus(ctx, sqlc.ListProjectsByStatusParams{
		Status: status,
		Limit:  limit,
		Offset: offset,
	})
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
	return r.queries.ListWBSTasks(ctx, sqlc.ListWBSTasksParams{
		Limit:  limit,
		Offset: offset,
	})
}

func (r *sqliteRepo) ListWBSTasksByProject(ctx context.Context, projectID, limit, offset int64) ([]sqlc.WbsTask, error) {
	return r.queries.ListWBSTasksByProject(ctx, sqlc.ListWBSTasksByProjectParams{
		ProjectID: projectID,
		Limit:     limit,
		Offset:    offset,
	})
}

func (r *sqliteRepo) ListWBSTasksByStatus(ctx context.Context, status string, limit, offset int64) ([]sqlc.WbsTask, error) {
	return r.queries.ListWBSTasksByStatus(ctx, sqlc.ListWBSTasksByStatusParams{
		Status: status,
		Limit:  limit,
		Offset: offset,
	})
}

func (r *sqliteRepo) UpdateWBSTask(ctx context.Context, arg sqlc.UpdateWBSTaskParams) (sqlc.WbsTask, error) {
	return r.queries.UpdateWBSTask(ctx, arg)
}

func (r *sqliteRepo) DeleteWBSTask(ctx context.Context, id int64) error {
	return r.queries.DeleteWBSTask(ctx, id)
}
