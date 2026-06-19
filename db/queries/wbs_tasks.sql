-- name: InsertWBSTask :one
INSERT INTO wbs_tasks (project_id, title, status, due_date)
VALUES (?, ?, ?, ?)
RETURNING *;

-- name: GetWBSTaskByID :one
SELECT * FROM wbs_tasks WHERE id = ? LIMIT 1;

-- name: ListWBSTasks :many
SELECT * FROM wbs_tasks
ORDER BY created_at DESC
LIMIT ? OFFSET ?;

-- name: ListWBSTasksByProject :many
SELECT * FROM wbs_tasks
WHERE project_id = ?
ORDER BY created_at DESC
LIMIT ? OFFSET ?;

-- name: ListWBSTasksByStatus :many
SELECT * FROM wbs_tasks
WHERE status = ?
ORDER BY created_at DESC
LIMIT ? OFFSET ?;

-- name: UpdateWBSTask :one
UPDATE wbs_tasks
SET title = ?, status = ?, due_date = ?
WHERE id = ?
RETURNING *;

-- name: DeleteWBSTask :exec
DELETE FROM wbs_tasks WHERE id = ?;
