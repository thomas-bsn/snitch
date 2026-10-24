-- +goose Up
-- In-flight links from older releases lack a session binding and are invalidated.
DELETE FROM oauth_states;
ALTER TABLE oauth_states ADD COLUMN session_hash bytea;
ALTER TABLE oauth_states ADD CONSTRAINT oauth_link_session CHECK ((user_id IS NULL) = (session_hash IS NULL));

-- +goose Down
ALTER TABLE oauth_states DROP COLUMN session_hash;
