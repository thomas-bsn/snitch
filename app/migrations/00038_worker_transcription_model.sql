-- +goose Up
ALTER TABLE workers ADD COLUMN transcription_model text NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE workers DROP COLUMN transcription_model;
