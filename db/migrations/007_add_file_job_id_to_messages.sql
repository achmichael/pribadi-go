ALTER TABLE web_chat_messages ADD COLUMN file_job_id TEXT REFERENCES web_upload_jobs(id) ON DELETE SET NULL;
