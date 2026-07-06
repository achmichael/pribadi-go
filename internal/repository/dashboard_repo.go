package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/achmichael/pribadi-go/internal/domain"
)

// DashboardRepository provides data access for dashboard-related tables.
type DashboardRepository interface {
	// Agent Config
	GetAgentConfigAll(ctx context.Context) (map[string]domain.AgentConfig, error)
	GetAgentConfigByKey(ctx context.Context, key string) (*domain.AgentConfig, error)
	UpsertAgentConfig(ctx context.Context, config domain.AgentConfig) error

	// Custom Entities
	ListEntitySchemas(ctx context.Context) ([]domain.CustomEntitySchema, error)
	GetEntitySchema(ctx context.Context, id string) (*domain.CustomEntitySchema, error)
	CreateEntitySchema(ctx context.Context, schema domain.CustomEntitySchema) error
	UpdateEntitySchema(ctx context.Context, schema domain.CustomEntitySchema) error
	DeleteEntitySchema(ctx context.Context, id string) error

	ListEntityRecords(ctx context.Context, schemaID string) ([]domain.CustomEntityRecord, error)
	GetEntityRecord(ctx context.Context, id string) (*domain.CustomEntityRecord, error)
	CreateEntityRecord(ctx context.Context, record domain.CustomEntityRecord) error
	UpdateEntityRecord(ctx context.Context, record domain.CustomEntityRecord) error
	DeleteEntityRecord(ctx context.Context, id string) error

	// Cron Jobs
	ListCronJobs(ctx context.Context) ([]domain.CronJob, error)
	GetCronJob(ctx context.Context, id string) (*domain.CronJob, error)
	CreateCronJob(ctx context.Context, job domain.CronJob) error
	UpdateCronJob(ctx context.Context, job domain.CronJob) error
	DeleteCronJob(ctx context.Context, id string) error
	LogCronJobExecution(ctx context.Context, log domain.CronJobLog) error
	GetCronJobLogs(ctx context.Context, jobID string, limit int) ([]domain.CronJobLog, error)

	// Stock Watchlist
	ListStockWatchlist(ctx context.Context) ([]domain.StockWatchlist, error)
	GetStockWatchlist(ctx context.Context, id string) (*domain.StockWatchlist, error)
	CreateStockWatchlist(ctx context.Context, stock domain.StockWatchlist) error
	UpdateStockWatchlist(ctx context.Context, stock domain.StockWatchlist) error
	DeleteStockWatchlist(ctx context.Context, id string) error
	UpdateStockPrice(ctx context.Context, ticker string, price float64) error

	// Users
	GetDashboardUserByUsername(ctx context.Context, username string) (*domain.DashboardUser, error)
	UpdateUserLastLogin(ctx context.Context, userID string) error
	CreateDashboardUser(ctx context.Context, user domain.DashboardUser) error
}

type dashboardRepo struct {
	db *sql.DB
}

// NewDashboardRepository creates a new instance of DashboardRepository
func NewDashboardRepository(db *sql.DB) DashboardRepository {
	return &dashboardRepo{db: db}
}

// ─── Agent Config ──────────────────────────────────────────────────────────

func (r *dashboardRepo) GetAgentConfigAll(ctx context.Context) (map[string]domain.AgentConfig, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT key, value_json, updated_at FROM agent_config`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	configs := make(map[string]domain.AgentConfig)
	for rows.Next() {
		var c domain.AgentConfig
		if err := rows.Scan(&c.Key, &c.ValueJSON, &c.UpdatedAt); err != nil {
			return nil, err
		}
		configs[c.Key] = c
	}
	return configs, rows.Err()
}

func (r *dashboardRepo) GetAgentConfigByKey(ctx context.Context, key string) (*domain.AgentConfig, error) {
	row := r.db.QueryRowContext(ctx, `SELECT key, value_json, updated_at FROM agent_config WHERE key = ?`, key)
	var c domain.AgentConfig
	if err := row.Scan(&c.Key, &c.ValueJSON, &c.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &c, nil
}

func (r *dashboardRepo) UpsertAgentConfig(ctx context.Context, config domain.AgentConfig) error {
	// Simple JSON validation
	if !json.Valid([]byte(config.ValueJSON)) {
		return fmt.Errorf("invalid JSON for config key %s", config.Key)
	}

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO agent_config (key, value_json, updated_at) 
		 VALUES (?, ?, CURRENT_TIMESTAMP)
		 ON CONFLICT(key) DO UPDATE SET 
			value_json = excluded.value_json,
			updated_at = CURRENT_TIMESTAMP`,
		config.Key, config.ValueJSON,
	)
	return err
}

// ─── Custom Entities Schemas ────────────────────────────────────────────────

func (r *dashboardRepo) ListEntitySchemas(ctx context.Context) ([]domain.CustomEntitySchema, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, name, label, description, fields_json, created_at, updated_at FROM custom_entity_schemas ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []domain.CustomEntitySchema
	for rows.Next() {
		var s domain.CustomEntitySchema
		if err := rows.Scan(&s.ID, &s.Name, &s.Label, &s.Description, &s.FieldsJSON, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		results = append(results, s)
	}
	return results, rows.Err()
}

func (r *dashboardRepo) GetEntitySchema(ctx context.Context, id string) (*domain.CustomEntitySchema, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, name, label, description, fields_json, created_at, updated_at FROM custom_entity_schemas WHERE id = ?`, id)
	var s domain.CustomEntitySchema
	if err := row.Scan(&s.ID, &s.Name, &s.Label, &s.Description, &s.FieldsJSON, &s.CreatedAt, &s.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

func (r *dashboardRepo) CreateEntitySchema(ctx context.Context, schema domain.CustomEntitySchema) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO custom_entity_schemas (id, name, label, description, fields_json) 
		 VALUES (?, ?, ?, ?, ?)`,
		schema.ID, schema.Name, schema.Label, schema.Description, schema.FieldsJSON,
	)
	return err
}

func (r *dashboardRepo) UpdateEntitySchema(ctx context.Context, schema domain.CustomEntitySchema) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE custom_entity_schemas 
		 SET name = ?, label = ?, description = ?, fields_json = ?, updated_at = CURRENT_TIMESTAMP
		 WHERE id = ?`,
		schema.Name, schema.Label, schema.Description, schema.FieldsJSON, schema.ID,
	)
	if err != nil {
		return err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *dashboardRepo) DeleteEntitySchema(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM custom_entity_schemas WHERE id = ?`, id)
	return err
}

// ─── Custom Entity Records ────────────────────────────────────────────────

func (r *dashboardRepo) ListEntityRecords(ctx context.Context, schemaID string) ([]domain.CustomEntityRecord, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, schema_id, data_json, created_at, updated_at FROM custom_entity_records WHERE schema_id = ? ORDER BY created_at DESC`, schemaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []domain.CustomEntityRecord
	for rows.Next() {
		var rec domain.CustomEntityRecord
		if err := rows.Scan(&rec.ID, &rec.SchemaID, &rec.DataJSON, &rec.CreatedAt, &rec.UpdatedAt); err != nil {
			return nil, err
		}
		results = append(results, rec)
	}
	return results, rows.Err()
}

func (r *dashboardRepo) GetEntityRecord(ctx context.Context, id string) (*domain.CustomEntityRecord, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, schema_id, data_json, created_at, updated_at FROM custom_entity_records WHERE id = ?`, id)
	var rec domain.CustomEntityRecord
	if err := row.Scan(&rec.ID, &rec.SchemaID, &rec.DataJSON, &rec.CreatedAt, &rec.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &rec, nil
}

func (r *dashboardRepo) CreateEntityRecord(ctx context.Context, record domain.CustomEntityRecord) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO custom_entity_records (id, schema_id, data_json) 
		 VALUES (?, ?, ?)`,
		record.ID, record.SchemaID, record.DataJSON,
	)
	return err
}

func (r *dashboardRepo) UpdateEntityRecord(ctx context.Context, record domain.CustomEntityRecord) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE custom_entity_records 
		 SET data_json = ?, updated_at = CURRENT_TIMESTAMP
		 WHERE id = ?`,
		record.DataJSON, record.ID,
	)
	if err != nil {
		return err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *dashboardRepo) DeleteEntityRecord(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM custom_entity_records WHERE id = ?`, id)
	return err
}

// ─── Cron Jobs ──────────────────────────────────────────────────────────────

func (r *dashboardRepo) ListCronJobs(ctx context.Context) ([]domain.CronJob, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, name, description, schedule_cron, job_type, config_json, target_whatsapp_jid, is_active, last_run_at, next_run_at, created_at, updated_at FROM cron_jobs ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []domain.CronJob
	for rows.Next() {
		var j domain.CronJob
		var lastRun, nextRun sql.NullTime
		if err := rows.Scan(&j.ID, &j.Name, &j.Description, &j.ScheduleCron, &j.JobType, &j.ConfigJSON, &j.TargetWhatsappJID, &j.IsActive, &lastRun, &nextRun, &j.CreatedAt, &j.UpdatedAt); err != nil {
			return nil, err
		}
		if lastRun.Valid {
			j.LastRunAt = &lastRun.Time
		}
		if nextRun.Valid {
			j.NextRunAt = &nextRun.Time
		}
		results = append(results, j)
	}
	return results, rows.Err()
}

func (r *dashboardRepo) GetCronJob(ctx context.Context, id string) (*domain.CronJob, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, name, description, schedule_cron, job_type, config_json, target_whatsapp_jid, is_active, last_run_at, next_run_at, created_at, updated_at FROM cron_jobs WHERE id = ?`, id)
	var j domain.CronJob
	var lastRun, nextRun sql.NullTime
	if err := row.Scan(&j.ID, &j.Name, &j.Description, &j.ScheduleCron, &j.JobType, &j.ConfigJSON, &j.TargetWhatsappJID, &j.IsActive, &lastRun, &nextRun, &j.CreatedAt, &j.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if lastRun.Valid {
		j.LastRunAt = &lastRun.Time
	}
	if nextRun.Valid {
		j.NextRunAt = &nextRun.Time
	}
	return &j, nil
}

func (r *dashboardRepo) CreateCronJob(ctx context.Context, job domain.CronJob) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO cron_jobs (id, name, description, schedule_cron, job_type, config_json, target_whatsapp_jid, is_active) 
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		job.ID, job.Name, job.Description, job.ScheduleCron, job.JobType, job.ConfigJSON, job.TargetWhatsappJID, job.IsActive,
	)
	return err
}

func (r *dashboardRepo) UpdateCronJob(ctx context.Context, job domain.CronJob) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE cron_jobs 
		 SET name = ?, description = ?, schedule_cron = ?, job_type = ?, config_json = ?, target_whatsapp_jid = ?, is_active = ?, updated_at = CURRENT_TIMESTAMP
		 WHERE id = ?`,
		job.Name, job.Description, job.ScheduleCron, job.JobType, job.ConfigJSON, job.TargetWhatsappJID, job.IsActive, job.ID,
	)
	if err != nil {
		return err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *dashboardRepo) DeleteCronJob(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM cron_jobs WHERE id = ?`, id)
	return err
}

func (r *dashboardRepo) LogCronJobExecution(ctx context.Context, log domain.CronJobLog) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO cron_job_logs (id, cron_job_id, status, output_message, error_detail) VALUES (?, ?, ?, ?, ?)`,
		log.ID, log.CronJobID, log.Status, log.OutputMessage, log.ErrorDetail,
	)
	// Update last_run_at in cron_jobs
	r.db.ExecContext(ctx, `UPDATE cron_jobs SET last_run_at = CURRENT_TIMESTAMP WHERE id = ?`, log.CronJobID)
	return err
}

func (r *dashboardRepo) GetCronJobLogs(ctx context.Context, jobID string, limit int) ([]domain.CronJobLog, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, cron_job_id, executed_at, status, output_message, error_detail FROM cron_job_logs WHERE cron_job_id = ? ORDER BY executed_at DESC LIMIT ?`, jobID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []domain.CronJobLog
	for rows.Next() {
		var l domain.CronJobLog
		if err := rows.Scan(&l.ID, &l.CronJobID, &l.ExecutedAt, &l.Status, &l.OutputMessage, &l.ErrorDetail); err != nil {
			return nil, err
		}
		results = append(results, l)
	}
	return results, rows.Err()
}

// ─── Stock Watchlist ──────────────────────────────────────────────────────

func (r *dashboardRepo) ListStockWatchlist(ctx context.Context) ([]domain.StockWatchlist, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, ticker, exchange, alert_condition, threshold_value, is_active, last_checked_price, last_checked_at, created_at, updated_at FROM stock_watchlist ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []domain.StockWatchlist
	for rows.Next() {
		var s domain.StockWatchlist
		var price sql.NullFloat64
		var checkedAt sql.NullTime
		if err := rows.Scan(&s.ID, &s.Ticker, &s.Exchange, &s.AlertCondition, &s.ThresholdValue, &s.IsActive, &price, &checkedAt, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		if price.Valid {
			s.LastCheckedPrice = &price.Float64
		}
		if checkedAt.Valid {
			s.LastCheckedAt = &checkedAt.Time
		}
		results = append(results, s)
	}
	return results, rows.Err()
}

func (r *dashboardRepo) GetStockWatchlist(ctx context.Context, id string) (*domain.StockWatchlist, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, ticker, exchange, alert_condition, threshold_value, is_active, last_checked_price, last_checked_at, created_at, updated_at FROM stock_watchlist WHERE id = ?`, id)
	var s domain.StockWatchlist
	var price sql.NullFloat64
	var checkedAt sql.NullTime
	if err := row.Scan(&s.ID, &s.Ticker, &s.Exchange, &s.AlertCondition, &s.ThresholdValue, &s.IsActive, &price, &checkedAt, &s.CreatedAt, &s.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if price.Valid {
		s.LastCheckedPrice = &price.Float64
	}
	if checkedAt.Valid {
		s.LastCheckedAt = &checkedAt.Time
	}
	return &s, nil
}

func (r *dashboardRepo) CreateStockWatchlist(ctx context.Context, stock domain.StockWatchlist) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO stock_watchlist (id, ticker, exchange, alert_condition, threshold_value, is_active) 
		 VALUES (?, ?, ?, ?, ?, ?)`,
		stock.ID, stock.Ticker, stock.Exchange, stock.AlertCondition, stock.ThresholdValue, stock.IsActive,
	)
	return err
}

func (r *dashboardRepo) UpdateStockWatchlist(ctx context.Context, stock domain.StockWatchlist) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE stock_watchlist 
		 SET ticker = ?, exchange = ?, alert_condition = ?, threshold_value = ?, is_active = ?, updated_at = CURRENT_TIMESTAMP
		 WHERE id = ?`,
		stock.Ticker, stock.Exchange, stock.AlertCondition, stock.ThresholdValue, stock.IsActive, stock.ID,
	)
	if err != nil {
		return err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *dashboardRepo) DeleteStockWatchlist(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM stock_watchlist WHERE id = ?`, id)
	return err
}

func (r *dashboardRepo) UpdateStockPrice(ctx context.Context, ticker string, price float64) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE stock_watchlist 
		 SET last_checked_price = ?, last_checked_at = CURRENT_TIMESTAMP 
		 WHERE ticker = ?`,
		price, ticker,
	)
	return err
}

// ─── Dashboard Users ──────────────────────────────────────────────────────

func (r *dashboardRepo) GetDashboardUserByUsername(ctx context.Context, username string) (*domain.DashboardUser, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, username, password_hash, created_at, last_login_at FROM dashboard_users WHERE username = ?`, username)
	var u domain.DashboardUser
	var lastLogin sql.NullTime
	if err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.CreatedAt, &lastLogin); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if lastLogin.Valid {
		u.LastLoginAt = &lastLogin.Time
	}
	return &u, nil
}

func (r *dashboardRepo) UpdateUserLastLogin(ctx context.Context, userID string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE dashboard_users SET last_login_at = CURRENT_TIMESTAMP WHERE id = ?`, userID)
	return err
}

func (r *dashboardRepo) CreateDashboardUser(ctx context.Context, user domain.DashboardUser) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO dashboard_users (id, username, password_hash) VALUES (?, ?, ?)`,
		user.ID, user.Username, user.PasswordHash,
	)
	return err
}
