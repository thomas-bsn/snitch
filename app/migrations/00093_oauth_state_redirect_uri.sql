-- +goose Up
-- The Discord return address is fixed when the sign-in starts and replayed at the callback.
-- In-flight states (10 minute lifetime) predate it and cannot be completed safely: invalidated.
DELETE FROM oauth_states;
ALTER TABLE oauth_states ADD COLUMN redirect_uri text NOT NULL;

-- +goose Down
ALTER TABLE oauth_states DROP COLUMN redirect_uri;
