-- +goose Up
ALTER TABLE users ADD COLUMN deleted_at timestamptz;
DROP INDEX users_username_lower;
CREATE UNIQUE INDEX users_username_lower ON users(lower(username)) WHERE deleted_at IS NULL;

-- +goose Down
UPDATE users SET username='deleted-' || id::text WHERE deleted_at IS NOT NULL;
DROP INDEX users_username_lower;
CREATE UNIQUE INDEX users_username_lower ON users(lower(username));
ALTER TABLE users DROP COLUMN deleted_at;
