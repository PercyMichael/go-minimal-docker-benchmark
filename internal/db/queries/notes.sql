-- name: ListNotesByUserID :many
SELECT id, user_id, title, content, tags, color, is_pinned, created_at, updated_at
FROM notes
WHERE user_id = $1
ORDER BY is_pinned DESC, updated_at DESC;

-- name: GetNoteByID :one
SELECT id, user_id, title, content, tags, color, is_pinned, created_at, updated_at
FROM notes
WHERE id = $1 AND user_id = $2;

-- name: CreateNote :one
INSERT INTO notes (user_id, title, content, tags, color, is_pinned)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, user_id, title, content, tags, color, is_pinned, created_at, updated_at;

-- name: DeleteNote :exec
DELETE FROM notes WHERE id = $1 AND user_id = $2;
