package domain

import (
	"context"
	"time"
)

type SourceType string

const (
	SourceHTTPAPI   SourceType = "http_api"
	SourceWebScrape SourceType = "web_scrape"
)

type ConditionMode string

const (
	ConditionStructural ConditionMode = "structural"
	ConditionNLJudge    ConditionMode = "nl_judge"
)

type StructuralOperator string

const (
	OpAbove         StructuralOperator = "above"
	OpBelow         StructuralOperator = "below"
	OpPercentChange StructuralOperator = "percent_change"
	OpEquals        StructuralOperator = "equals"
	OpContains      StructuralOperator = "contains"
)

type HTTPSourceConfig struct {
	URL           string            `json:"url"`
	Method        string            `json:"method"`
	Headers       map[string]string `json:"headers,omitempty"`
	AuthType      string            `json:"auth_type,omitempty"`
	AuthValue     string            `json:"auth_value,omitempty"`
	Body          string            `json:"body,omitempty"`
	ExtractPath   string            `json:"extract_path"`
	ExtractAsText bool              `json:"extract_as_text"`
}

type ScrapeSourceConfig struct {
	URL         string `json:"url"`
	Description string `json:"description"`
}

type MonitorTask struct {
	ID           int64      `json:"id"`
	UserID       string     `json:"user_id"`
	Platform     string     `json:"platform"`
	Category     string     `json:"category,omitempty"`
	Name         string     `json:"name"`
	SourceType   SourceType `json:"source_type"`
	SourceConfig string     `json:"source_config"`

	ConditionMode  ConditionMode      `json:"condition_mode"`
	Operator       StructuralOperator `json:"operator,omitempty"`
	ConditionValue float64            `json:"condition_value,omitempty"`
	ConditionText  string             `json:"condition_text,omitempty"`
	ConditionPrompt string            `json:"condition_prompt,omitempty"`

	PollIntervalSeconds int `json:"poll_interval_seconds"`

	LastValue      *float64   `json:"last_value,omitempty"`
	LastValueText  *string    `json:"last_value_text,omitempty"`
	LastCheckedAt  *time.Time `json:"last_checked_at,omitempty"`
	LastNotifiedAt *time.Time `json:"last_notified_at,omitempty"`
	Status         string     `json:"status"`
	CreatedAt      time.Time  `json:"created_at"`
}

type ExtractionRule struct {
	ID            int64     `json:"id"`
	MonitorTaskID int64     `json:"monitor_task_id"`
	Selector      string    `json:"selector"`
	RuleType      string    `json:"rule_type"`
	DerivedAt     time.Time `json:"derived_at"`
	FailCount     int       `json:"fail_count"`
}

type DataSourceProvider interface {
	Fetch(ctx context.Context, task MonitorTask) (valueNumeric *float64, valueText string, rawSnapshot string, err error)
}

type ConditionEvaluator interface {
	Evaluate(ctx context.Context, task MonitorTask, currentNumeric *float64, currentText string, previousNumeric *float64, rawSnapshot string) (bool, error)
}

type MonitorTaskRepository interface {
	ListActive(ctx context.Context) ([]MonitorTask, error)
	GetByID(ctx context.Context, id int64) (*MonitorTask, error)
	ListByUser(ctx context.Context, userID string, category string) ([]MonitorTask, error)
	Create(ctx context.Context, task MonitorTask) (int64, error)
	Update(ctx context.Context, task MonitorTask) error
	Delete(ctx context.Context, id int64) error
	UpdateLastValue(ctx context.Context, id int64, numeric *float64, text string, checkedAt time.Time) error
	UpdateLastNotified(ctx context.Context, id int64, t time.Time) error
	MarkStatus(ctx context.Context, id int64, status string) error
}

type ExtractionRuleRepository interface {
	GetByTaskID(ctx context.Context, taskID int64) (*ExtractionRule, error)
	Save(ctx context.Context, rule ExtractionRule) error
	IncrementFailCount(ctx context.Context, taskID int64) error
}

type NLJudgeBudgetRepository interface {
	GetCallCount(ctx context.Context, taskID int64, windowStart time.Time) (int, error)
	IncrementCallCount(ctx context.Context, taskID int64) error
	GetDailyLimit(ctx context.Context, taskID int64) (int, error)
}

type MonitorValueHistoryRepository interface {
	GetLastSnapshotHash(ctx context.Context, taskID int64) (string, error)
	Insert(ctx context.Context, taskID int64, value *float64, valueText string, snapshotHash string) error
}
