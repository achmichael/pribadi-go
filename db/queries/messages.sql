-- name: InsertMessage :one
INSERT INTO messages (wa_id, from_jid, to_jid, content, media_type, timestamp)
VALUES (?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetMessageByID :one
SELECT * FROM messages WHERE id = ? LIMIT 1;

-- name: GetMessageByWAID :one
SELECT * FROM messages WHERE wa_id = ? LIMIT 1;

-- name: ListMessages :many
SELECT * FROM messages
ORDER BY timestamp DESC
LIMIT ? OFFSET ?;

-- name: ListMessagesByJID :many
SELECT * FROM messages
WHERE from_jid = ? OR to_jid = ?
ORDER BY timestamp DESC
LIMIT ? OFFSET ?;

-- name: UpdateMessage :one
UPDATE messages
SET content = ?, media_type = ?
WHERE id = ?
RETURNING *;

-- name: DeleteMessage :exec
DELETE FROM messages WHERE id = ?;
