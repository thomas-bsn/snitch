-- +goose Up
-- Display name of the linked Discord identity (global_name, else username), read-only and shown on the
-- profile. Nullable: accounts linked before this migration stay without a name until they link again.
-- No OAuth token or other Discord data is ever stored.
ALTER TABLE users ADD COLUMN discord_name text;
-- +goose Down
ALTER TABLE users DROP COLUMN discord_name;
