-- +goose Up
CREATE TABLE account_avatars (
 user_id bigint PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
 png bytea NOT NULL CHECK (octet_length(png) <= 1048576)
);
-- +goose Down
DROP TABLE account_avatars;
