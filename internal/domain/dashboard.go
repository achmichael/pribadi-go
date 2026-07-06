package domain

import (
	"time"
)

// AgentConfig represents a single configuration key-value pair for the AI Agent
type AgentConfig struct {
	Key       string    `json:"key"`
	ValueJSON string    `json:"value_json"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CustomEntitySchema defines the structure of a user-defined entity
type CustomEntitySchema struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Label       string    `json:"label"`
	Description string    `json:"description"`
	FieldsJSON  string    `json:"fields_json"` // JSON array of field definitions
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CustomEntityRecord represents a single record of a user-defined entity
type CustomEntityRecord struct {
	ID        string    `json:"id"`
	SchemaID  string    `json:"schema_id"`
	DataJSON  string    `json:"data_json"` // JSON object conforming to schema's fields_json
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CronJob represents a scheduled task
type CronJob struct {
	ID                string    `json:"id"`
	Name              string    `json:"name"`
	Description       string    `json:"description"`
	ScheduleCron      string    `json:"schedule_cron"`
	JobType           string    `json:"job_type"` // 'stock_alert', 'custom_reminder', 'entity_digest', 'custom_prompt'
	ConfigJSON        string    `json:"config_json"`
	TargetWhatsappJID string    `json:"target_whatsapp_jid"`
	IsActive          bool      `json:"is_active"`
	LastRunAt         *time.Time `json:"last_run_at,omitempty"`
	NextRunAt         *time.Time `json:"next_run_at,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// CronJobLog represents an execution log of a CronJob
type CronJobLog struct {
	ID            string    `json:"id"`
	CronJobID     string    `json:"cron_job_id"`
	ExecutedAt    time.Time `json:"executed_at"`
	Status        string    `json:"status"` // 'success', 'failed'
	OutputMessage string    `json:"output_message,omitempty"`
	ErrorDetail   string    `json:"error_detail,omitempty"`
}

// StockWatchlist represents a stock ticker being monitored
type StockWatchlist struct {
	ID               string    `json:"id"`
	Ticker           string    `json:"ticker"`
	Exchange         string    `json:"exchange"` // e.g. IDX
	AlertCondition   string    `json:"alert_condition"` // 'above', 'below', 'percent_change'
	ThresholdValue   float64   `json:"threshold_value"`
	IsActive         bool      `json:"is_active"`
	LastCheckedPrice *float64  `json:"last_checked_price,omitempty"`
	LastCheckedAt    *time.Time `json:"last_checked_at,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// DashboardUser represents a user allowed to access the dashboard
type DashboardUser struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"` // never expose in JSON
	CreatedAt    time.Time `json:"created_at"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty"`
}
