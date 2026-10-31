-- +goose Up
CREATE TABLE transcription_settings (
    id boolean PRIMARY KEY DEFAULT true CHECK (id),
    mode text NOT NULL CHECK (mode IN ('enabled', 'disabled')),
    endpoint text NOT NULL,
    version text NOT NULL,
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE transcription_settings;
