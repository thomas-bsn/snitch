-- +goose Up
ALTER TABLE users ADD COLUMN display_name text CHECK (char_length(display_name) BETWEEN 1 AND 100);
ALTER TABLE users ADD COLUMN discord_username text;

-- +goose Down
ALTER TABLE users DROP COLUMN discord_username;
ALTER TABLE users DROP COLUMN display_name;
