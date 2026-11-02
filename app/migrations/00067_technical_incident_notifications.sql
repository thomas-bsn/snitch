-- +goose Up
-- Technical incident notifications (administrators): one active notification per cause
-- (kind + cause), refreshed on every retry and resolved by itself when the cause is gone.
-- The payload only carries codes: no secret, path or stack trace.
ALTER TABLE notifications DROP CONSTRAINT notifications_kind_check;
ALTER TABLE notifications ADD CONSTRAINT notifications_kind_check CHECK (kind IN ('needs_speaker_review', 'worker_unavailable', 'provider_unavailable', 'model_missing', 'job_failed'));
ALTER TABLE notifications ADD COLUMN cause text NOT NULL DEFAULT '';
ALTER TABLE notifications ADD COLUMN job_id bigint;
ALTER TABLE notifications ADD COLUMN step text NOT NULL DEFAULT '';
ALTER TABLE notifications ADD COLUMN error_code text NOT NULL DEFAULT '';
DROP INDEX notifications_one_active;
CREATE UNIQUE INDEX notifications_one_active ON notifications (kind, clip_id) WHERE state = 'active' AND kind = 'needs_speaker_review';
CREATE UNIQUE INDEX notifications_one_active_cause ON notifications (kind, cause) WHERE state = 'active' AND cause <> '';
-- +goose Down
-- Lossy: incident notifications are deleted.
DELETE FROM notifications WHERE kind <> 'needs_speaker_review';
DROP INDEX notifications_one_active_cause;
DROP INDEX notifications_one_active;
CREATE UNIQUE INDEX notifications_one_active ON notifications (kind, clip_id) WHERE state = 'active';
ALTER TABLE notifications DROP COLUMN error_code;
ALTER TABLE notifications DROP COLUMN step;
ALTER TABLE notifications DROP COLUMN job_id;
ALTER TABLE notifications DROP COLUMN cause;
ALTER TABLE notifications DROP CONSTRAINT notifications_kind_check;
ALTER TABLE notifications ADD CONSTRAINT notifications_kind_check CHECK (kind IN ('needs_speaker_review'));
