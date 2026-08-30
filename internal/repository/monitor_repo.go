package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/achmichael/pribadi-go/internal/domain"
)

type MonitorRepository interface {
	domain.MonitorTaskRepository
	domain.ExtractionRuleRepository
	domain.NLJudgeBudgetRepository
	domain.MonitorValueHistoryRepository
}

type monitorRepo struct {
	db *sql.DB
}

func NewMonitorRepository(db *sql.DB) MonitorRepository {
	return &monitorRepo{db: db}
}

func (r *monitorRepo) ListActive(ctx context.Context) ([]domain.MonitorTask, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, user_id, platform, category, name, source_type, source_config,
		        condition_mode, operator, condition_value, condition_text, condition_prompt,
		        poll_interval_seconds, last_value, last_value_text, last_checked_at,
		        last_notified_at, status, created_at
		 FROM monitor_tasks WHERE status = 'active'`)
	if err != nil {
		return nil, err
	}

	fmt.Print("rows", rows)
	defer rows.Close()
	return scanMonitorTasks(rows)
}

func (r *monitorRepo) GetByID(ctx context.Context, id int64) (*domain.MonitorTask, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, user_id, platform, category, name, source_type, source_config,
		        condition_mode, operator, condition_value, condition_text, condition_prompt,
		        poll_interval_seconds, last_value, last_value_text, last_checked_at,
		        last_notified_at, status, created_at
		 FROM monitor_tasks WHERE id = ?`, id)
	return scanMonitorTask(row)
}

func (r *monitorRepo) ListByUser(ctx context.Context, userID string, category string) ([]domain.MonitorTask, error) {
	q := `SELECT id, user_id, platform, category, name, source_type, source_config,
	             condition_mode, operator, condition_value, condition_text, condition_prompt,
	             poll_interval_seconds, last_value, last_value_text, last_checked_at,
	             last_notified_at, status, created_at
	      FROM monitor_tasks WHERE user_id = ?`
	args := []any{userID}
	if category != "" {
		q += ` AND category = ?`
		args = append(args, category)
	}
	q += ` ORDER BY created_at DESC`
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tasks, err := scanMonitorTasks(rows)
	if err == nil {
		fmt.Printf("[MonitorRepo] ListByUser: userID=%s, category=%s, rows=%d\n", userID, category, len(tasks))
	}
	return tasks, err
}

func (r *monitorRepo) Create(ctx context.Context, task domain.MonitorTask) (int64, error) {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO monitor_tasks (user_id, platform, category, name, source_type, source_config,
		 condition_mode, operator, condition_value, condition_text, condition_prompt,
		 poll_interval_seconds, status)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		task.UserID, task.Platform, task.Category, task.Name, task.SourceType, task.SourceConfig,
		task.ConditionMode, task.Operator, task.ConditionValue, task.ConditionText, task.ConditionPrompt,
		task.PollIntervalSeconds, "active")
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r *monitorRepo) Update(ctx context.Context, task domain.MonitorTask) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE monitor_tasks SET category = ?, name = ?, source_type = ?, source_config = ?,
		 condition_mode = ?, operator = ?, condition_value = ?, condition_text = ?,
		 condition_prompt = ?, poll_interval_seconds = ?
		 WHERE id = ?`,
		task.Category, task.Name, task.SourceType, task.SourceConfig,
		task.ConditionMode, task.Operator, task.ConditionValue, task.ConditionText,
		task.ConditionPrompt, task.PollIntervalSeconds, task.ID)
	return err
}

func (r *monitorRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM monitor_tasks WHERE id = ?`, id)
	return err
}

func (r *monitorRepo) UpdateLastValue(ctx context.Context, id int64, numeric *float64, text string, checkedAt time.Time) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE monitor_tasks SET last_value = ?, last_value_text = ?, last_checked_at = ? WHERE id = ?`,
		numeric, text, checkedAt, id)
	return err
}

func (r *monitorRepo) UpdateLastNotified(ctx context.Context, id int64, t time.Time) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE monitor_tasks SET last_notified_at = ? WHERE id = ?`, t, id)
	return err
}

func (r *monitorRepo) MarkStatus(ctx context.Context, id int64, status string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE monitor_tasks SET status = ? WHERE id = ?`, status, id)
	return err
}

func (r *monitorRepo) GetByTaskID(ctx context.Context, taskID int64) (*domain.ExtractionRule, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, monitor_task_id, selector, rule_type, derived_at, fail_count
		 FROM monitor_extraction_rules WHERE monitor_task_id = ? ORDER BY id DESC LIMIT 1`, taskID)
	var rule domain.ExtractionRule
	if err := row.Scan(&rule.ID, &rule.MonitorTaskID, &rule.Selector, &rule.RuleType, &rule.DerivedAt, &rule.FailCount); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &rule, nil
}

func (r *monitorRepo) Save(ctx context.Context, rule domain.ExtractionRule) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO monitor_extraction_rules (monitor_task_id, selector, rule_type, derived_at)
		 VALUES (?, ?, ?, ?)`,
		rule.MonitorTaskID, rule.Selector, rule.RuleType, rule.DerivedAt)
	return err
}

func (r *monitorRepo) IncrementFailCount(ctx context.Context, taskID int64) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE monitor_extraction_rules SET fail_count = fail_count + 1
		 WHERE monitor_task_id = ? ORDER BY id DESC LIMIT 1`, taskID)
	return err
}

func (r *monitorRepo) GetCallCount(ctx context.Context, taskID int64, windowStart time.Time) (int, error) {
	var count int
	var windowStarted sql.NullTime
	err := r.db.QueryRowContext(ctx,
		`SELECT calls_in_window, window_started_at FROM monitor_nl_judge_budget WHERE monitor_task_id = ?`,
		taskID).Scan(&count, &windowStarted)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	if windowStarted.Valid && windowStarted.Time.Before(windowStart) {
		r.db.ExecContext(ctx,
			`UPDATE monitor_nl_judge_budget SET calls_in_window = 0, window_started_at = ? WHERE monitor_task_id = ?`,
			windowStart, taskID)
		return 0, nil
	}
	return count, nil
}

func (r *monitorRepo) IncrementCallCount(ctx context.Context, taskID int64) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO monitor_nl_judge_budget (monitor_task_id, calls_in_window, window_started_at)
		 VALUES (?, 1, CURRENT_TIMESTAMP)
		 ON CONFLICT(monitor_task_id) DO UPDATE SET calls_in_window = monitor_nl_judge_budget.calls_in_window + 1`,
		taskID)
	return err
}

func (r *monitorRepo) GetDailyLimit(ctx context.Context, taskID int64) (int, error) {
	var limit int
	err := r.db.QueryRowContext(ctx,
		`SELECT daily_limit FROM monitor_nl_judge_budget WHERE monitor_task_id = ?`, taskID).Scan(&limit)
	if err == sql.ErrNoRows {
		return 100, nil
	}
	return limit, err
}

func (r *monitorRepo) GetLastSnapshotHash(ctx context.Context, taskID int64) (string, error) {
	var hash sql.NullString
	err := r.db.QueryRowContext(ctx,
		`SELECT raw_snapshot_hash FROM monitor_value_history WHERE monitor_task_id = ? ORDER BY recorded_at DESC LIMIT 1`,
		taskID).Scan(&hash)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return hash.String, nil
}

func (r *monitorRepo) Insert(ctx context.Context, taskID int64, value *float64, valueText string, snapshotHash string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO monitor_value_history (monitor_task_id, value, value_text, raw_snapshot_hash)
		 VALUES (?, ?, ?, ?)`,
		taskID, value, valueText, snapshotHash)
	return err
}

func scanMonitorTask(row *sql.Row) (*domain.MonitorTask, error) {
	var t domain.MonitorTask
	var category, operator, condText, condPrompt sql.NullString
	var condValue sql.NullFloat64
	var lastValue sql.NullFloat64
	var lastValueText sql.NullString
	var lastChecked, lastNotified sql.NullTime
	if err := row.Scan(
		&t.ID, &t.UserID, &t.Platform, &category, &t.Name, &t.SourceType, &t.SourceConfig,
		&t.ConditionMode, &operator, &condValue, &condText, &condPrompt,
		&t.PollIntervalSeconds, &lastValue, &lastValueText, &lastChecked,
		&lastNotified, &t.Status, &t.CreatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	t.Category = category.String
	t.Operator = domain.StructuralOperator(operator.String)
	t.ConditionValue = condValue.Float64
	t.ConditionText = condText.String
	t.ConditionPrompt = condPrompt.String
	if lastValue.Valid {
		t.LastValue = &lastValue.Float64
	}
	if lastValueText.Valid {
		t.LastValueText = &lastValueText.String
	}
	if lastChecked.Valid {
		t.LastCheckedAt = &lastChecked.Time
	}
	if lastNotified.Valid {
		t.LastNotifiedAt = &lastNotified.Time
	}
	return &t, nil
}

func scanMonitorTasks(rows *sql.Rows) ([]domain.MonitorTask, error) {
	var results []domain.MonitorTask
	for rows.Next() {
		var t domain.MonitorTask
		var category, operator, condText, condPrompt sql.NullString
		var condValue sql.NullFloat64
		var lastValue sql.NullFloat64
		var lastValueText sql.NullString
		var lastChecked, lastNotified sql.NullTime
		if err := rows.Scan(
			&t.ID, &t.UserID, &t.Platform, &category, &t.Name, &t.SourceType, &t.SourceConfig,
			&t.ConditionMode, &operator, &condValue, &condText, &condPrompt,
			&t.PollIntervalSeconds, &lastValue, &lastValueText, &lastChecked,
			&lastNotified, &t.Status, &t.CreatedAt,
		); err != nil {
			return nil, err
		}
		t.Category = category.String
		t.Operator = domain.StructuralOperator(operator.String)
		t.ConditionValue = condValue.Float64
		t.ConditionText = condText.String
		t.ConditionPrompt = condPrompt.String
		if lastValue.Valid {
			t.LastValue = &lastValue.Float64
		}
		if lastValueText.Valid {
			t.LastValueText = &lastValueText.String
		}
		if lastChecked.Valid {
			t.LastCheckedAt = &lastChecked.Time
		}
		if lastNotified.Valid {
			t.LastNotifiedAt = &lastNotified.Time
		}
		results = append(results, t)
	}
	return results, rows.Err()
}
