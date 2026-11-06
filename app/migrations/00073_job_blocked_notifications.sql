-- +goose Up
-- A pending job that can never start until someone acts (incompatible or unvalidated transcription configuration)
-- is an administrator incident of its own kind, grouped by cause code.
ALTER TABLE notifications DROP CONSTRAINT notifications_kind_check;
ALTER TABLE notifications ADD CONSTRAINT notifications_kind_check CHECK (kind IN ('needs_speaker_review', 'worker_unavailable', 'provider_unavailable', 'model_missing', 'job_failed', 'job_blocked'));
-- +goose Down
DELETE FROM notifications WHERE kind = 'job_blocked';
ALTER TABLE notifications DROP CONSTRAINT notifications_kind_check;
ALTER TABLE notifications ADD CONSTRAINT notifications_kind_check CHECK (kind IN ('needs_speaker_review', 'worker_unavailable', 'provider_unavailable', 'model_missing', 'job_failed'));
