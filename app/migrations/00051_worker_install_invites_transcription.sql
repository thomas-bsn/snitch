-- +goose Up
ALTER TABLE worker_install_invites
    ADD COLUMN transcription_mode text NOT NULL DEFAULT 'unconfigured',
    ADD COLUMN transcription_endpoint text NOT NULL DEFAULT '',
    ADD COLUMN transcription_model text NOT NULL DEFAULT '',
    ADD COLUMN transcription_language text NOT NULL DEFAULT '';
-- +goose Down
ALTER TABLE worker_install_invites
    DROP COLUMN transcription_mode,
    DROP COLUMN transcription_endpoint,
    DROP COLUMN transcription_model,
    DROP COLUMN transcription_language;