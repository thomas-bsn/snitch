-- name: GetUser :one
SELECT id, username, role, approved FROM users WHERE id = $1;

-- name: ListPending :many
SELECT id, username, role, approved FROM users WHERE NOT approved ORDER BY id;

-- name: ApproveUser :execrows
UPDATE users SET approved = true WHERE id = $1 AND NOT approved;
