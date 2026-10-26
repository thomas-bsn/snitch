-- +goose Up
-- Interface language chosen by the account holder. NULL means no choice was made yet:
-- the browser's remembered choice, then English, applies. Only supported codes are stored.
ALTER TABLE users ADD COLUMN language text CHECK (language IN ('en','fr'));

-- +goose Down
ALTER TABLE users DROP COLUMN language;
