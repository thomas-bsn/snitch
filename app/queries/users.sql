-- name: GetUser :one
SELECT id, username, role, approved, language, COALESCE(display_name, CASE WHEN password_hash IS NULL AND starts_with(username,'discord_') THEN discord_name END, username)::text AS display_name FROM users WHERE id = $1;

-- name: ListPending :many
SELECT id, username, role, approved FROM users WHERE NOT approved AND deleted_at IS NULL ORDER BY id;

-- name: ApproveUser :execrows
UPDATE users SET approved = true WHERE id = $1 AND NOT approved AND deleted_at IS NULL;
