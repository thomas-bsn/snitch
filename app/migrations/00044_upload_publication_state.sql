-- +goose Up
ALTER TABLE upload_destinations ADD COLUMN attempts integer NOT NULL DEFAULT 0;
ALTER TABLE upload_destinations ADD COLUMN last_error text;
-- +goose Down
ALTER TABLE upload_destinations DROP COLUMN last_error;
ALTER TABLE upload_destinations DROP COLUMN attempts;
