-- name: InsertReminder :one
INSERT INTO reminders (jid, message, scheduled_at)
VALUES (?, ?, ?)
RETURNING *;

-- name: GetReminderByID :one
SELECT * FROM reminders WHERE id = ? LIMIT 1;

-- name: ListReminders :many
SELECT * FROM reminders
ORDER BY scheduled_at ASC
LIMIT ? OFFSET ?;

-- name: ListRemindersByJID :many
SELECT * FROM reminders
WHERE jid = ?
ORDER BY scheduled_at ASC
LIMIT ? OFFSET ?;

-- name: ListPendingReminders :many
SELECT * FROM reminders
WHERE is_sent = 0 AND scheduled_at <= ?
ORDER BY scheduled_at ASC;

-- name: UpdateReminder :one
UPDATE reminders
SET message = ?, scheduled_at = ?
WHERE id = ?
RETURNING *;

-- name: MarkReminderAsSent :exec
UPDATE reminders
SET is_sent = 1
WHERE id = ?;

-- name: DeleteReminder :exec
DELETE FROM reminders WHERE id = ?;
