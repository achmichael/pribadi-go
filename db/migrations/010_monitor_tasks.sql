CREATE TABLE IF NOT EXISTS monitor_tasks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    platform TEXT NOT NULL,
    category TEXT,
    name TEXT NOT NULL,
    source_type TEXT NOT NULL,
    source_config TEXT NOT NULL,
    condition_mode TEXT NOT NULL,
    operator TEXT,
    condition_value REAL,
    condition_text TEXT,
    condition_prompt TEXT,
    poll_interval_seconds INTEGER NOT NULL DEFAULT 300,
    last_value REAL,
    last_value_text TEXT,
    last_checked_at DATETIME,
    last_notified_at DATETIME,
    status TEXT NOT NULL DEFAULT 'active',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_monitor_tasks_user_id ON monitor_tasks(user_id);
CREATE INDEX IF NOT EXISTS idx_monitor_tasks_status ON monitor_tasks(status);

CREATE TABLE IF NOT EXISTS monitor_extraction_rules (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    monitor_task_id INTEGER NOT NULL REFERENCES monitor_tasks(id) ON DELETE CASCADE,
    selector TEXT NOT NULL,
    rule_type TEXT NOT NULL,
    derived_at DATETIME NOT NULL,
    fail_count INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_monitor_extraction_rules_task ON monitor_extraction_rules(monitor_task_id);

CREATE TABLE IF NOT EXISTS monitor_nl_judge_budget (
    monitor_task_id INTEGER PRIMARY KEY REFERENCES monitor_tasks(id) ON DELETE CASCADE,
    calls_in_window INTEGER NOT NULL DEFAULT 0,
    window_started_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    daily_limit INTEGER NOT NULL DEFAULT 100
);

CREATE TABLE IF NOT EXISTS monitor_value_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    monitor_task_id INTEGER NOT NULL REFERENCES monitor_tasks(id) ON DELETE CASCADE,
    value REAL,
    value_text TEXT,
    raw_snapshot_hash TEXT,
    recorded_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_monitor_value_history_task ON monitor_value_history(monitor_task_id);
