-- name: InsertProject :one
INSERT INTO projects (name, owner_jid, status)
VALUES (?, ?, ?)
RETURNING *;

-- name: GetProjectByID :one
SELECT * FROM projects WHERE id = ? LIMIT 1;

-- name: ListProjects :many
SELECT * FROM projects
ORDER BY created_at DESC
LIMIT ? OFFSET ?;

-- name: ListProjectsByOwner :many
SELECT * FROM projects
WHERE owner_jid = ?
ORDER BY created_at DESC
LIMIT ? OFFSET ?;

-- name: ListProjectsByStatus :many
SELECT * FROM projects
WHERE status = ?
ORDER BY created_at DESC
LIMIT ? OFFSET ?;

-- name: UpdateProject :one
UPDATE projects
SET name = ?, status = ?
WHERE id = ?
RETURNING *;

-- name: DeleteProject :exec
DELETE FROM projects WHERE id = ?;
