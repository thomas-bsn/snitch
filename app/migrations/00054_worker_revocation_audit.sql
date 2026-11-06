-- +goose Up
-- Minimal deletion audit kept on the revoked identity: name and revoked_at already exist, the reason is optional.
ALTER TABLE workers ADD COLUMN revoke_reason text NOT NULL DEFAULT '' CHECK (length(revoke_reason) <= 200);
-- Set when the local launcher confirmed the revocation (it then stops the service). Never a claim of local erasure.
ALTER TABLE workers ADD COLUMN revocation_acknowledged_at timestamptz;
ALTER TABLE job_events DROP CONSTRAINT job_events_event_check;
ALTER TABLE job_events ADD CONSTRAINT job_events_event_check CHECK (event IN ('attempt_started','retry_scheduled','marked_dead','admin_relaunched','cancelled','completed','lease_expired','worker_revoked'));
-- +goose Down
UPDATE job_events SET event='lease_expired' WHERE event='worker_revoked';
ALTER TABLE job_events DROP CONSTRAINT job_events_event_check;
ALTER TABLE job_events ADD CONSTRAINT job_events_event_check CHECK (event IN ('attempt_started','retry_scheduled','marked_dead','admin_relaunched','cancelled','completed','lease_expired'));
ALTER TABLE workers DROP COLUMN revocation_acknowledged_at;
ALTER TABLE workers DROP COLUMN revoke_reason;
