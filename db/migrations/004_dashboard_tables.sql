CREATE TABLE IF NOT EXISTS agent_config (
  key TEXT PRIMARY KEY,
  value_json TEXT NOT NULL,
  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS custom_entity_schemas (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  label TEXT NOT NULL,
  description TEXT,
  fields_json TEXT NOT NULL,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS custom_entity_records (
  id TEXT PRIMARY KEY,
  schema_id TEXT NOT NULL,
  data_json TEXT NOT NULL,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (schema_id) REFERENCES custom_entity_schemas(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_cer_schema ON custom_entity_records(schema_id);

CREATE TABLE IF NOT EXISTS cron_jobs (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  description TEXT,
  schedule_cron TEXT NOT NULL,
  job_type TEXT NOT NULL,
  config_json TEXT,
  target_whatsapp_jid TEXT,
  is_active BOOLEAN DEFAULT 1,
  last_run_at DATETIME,
  next_run_at DATETIME,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS cron_job_logs (
  id TEXT PRIMARY KEY,
  cron_job_id TEXT NOT NULL,
  executed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  status TEXT NOT NULL,
  output_message TEXT,
  error_detail TEXT,
  FOREIGN KEY (cron_job_id) REFERENCES cron_jobs(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS stock_watchlist (
  id TEXT PRIMARY KEY,
  ticker TEXT NOT NULL,
  exchange TEXT,
  alert_condition TEXT NOT NULL,
  threshold_value REAL NOT NULL,
  is_active BOOLEAN DEFAULT 1,
  last_checked_price REAL,
  last_checked_at DATETIME,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS dashboard_users (
  id TEXT PRIMARY KEY,
  username TEXT UNIQUE NOT NULL,
  password_hash TEXT NOT NULL,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  last_login_at DATETIME
);

-- Default configuration for agent_config
INSERT OR IGNORE INTO agent_config (key, value_json) VALUES 
('persona_name', '"Assistant"'),
('persona_description', '"Saya adalah asisten AI yang ramah dan cerdas."'),
('system_prompt_template', '"Anda adalah asisten AI. Identitas Anda: {{persona_name}} ({{persona_description}}). Gaya bahasa: {{tone_preference}}. Anda dihubungi melalui WhatsApp.\nBerikut adalah konteks fakta terkait pengguna:\n{{user_facts}}\n\nJika ada dokumen yang sedang dibahas:\n{{active_document_metadata}}"'),
('tone_preference', '"casual"'),
('language_default', '"id"'),
('feature_toggles', '{"rag": true, "cron": true, "stock": true}');
