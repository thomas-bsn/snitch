-- +goose Up
-- Demande d'accès (#283): an Inscription en attente raises one persistent notification for the administrators.
-- One logical event per account, whatever the retries: the unique index refuses a second one. The notification
-- resolves itself when the account is approved (by any path) or retired (a refusal), with the outcome.
ALTER TABLE notifications DROP CONSTRAINT notifications_kind_check;
ALTER TABLE notifications ADD CONSTRAINT notifications_kind_check CHECK (kind IN ('needs_speaker_review', 'worker_unavailable', 'provider_unavailable', 'model_missing', 'job_failed', 'job_blocked', 'folder_request_decided', 'account_request'));
ALTER TABLE notifications ADD COLUMN account_id bigint REFERENCES users(id) ON DELETE CASCADE;
ALTER TABLE notifications ADD COLUMN account_name text NOT NULL DEFAULT '';
ALTER TABLE notifications ADD COLUMN account_outcome text NOT NULL DEFAULT '' CHECK (account_outcome IN ('', 'approved', 'refused'));
CREATE UNIQUE INDEX notifications_one_account_request ON notifications (account_id) WHERE kind = 'account_request';

-- +goose StatementBegin
CREATE FUNCTION account_request_resolved() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF NEW.approved AND NOT OLD.approved THEN
  UPDATE notifications SET state='resolved',resolved_at=now(),updated_at=now(),account_outcome='approved'
   WHERE kind='account_request' AND account_id=NEW.id AND state='active';
 ELSIF NEW.deleted_at IS NOT NULL AND OLD.deleted_at IS NULL THEN
  -- A retired account is anonymized: its request forgets the name too.
  UPDATE notifications SET state=CASE WHEN state='active' THEN 'resolved' ELSE state END,resolved_at=COALESCE(resolved_at,now()),updated_at=now(),
   account_outcome=CASE WHEN state='active' THEN 'refused' ELSE account_outcome END,account_name=''
   WHERE kind='account_request' AND account_id=NEW.id;
 END IF;
 RETURN NULL;
END $$;
-- +goose StatementEnd
CREATE TRIGGER users_account_request_resolved AFTER UPDATE OF approved, deleted_at ON users FOR EACH ROW EXECUTE FUNCTION account_request_resolved();

-- +goose Down
DROP TRIGGER users_account_request_resolved ON users;
DROP FUNCTION account_request_resolved();
DELETE FROM notifications WHERE kind = 'account_request';
DROP INDEX notifications_one_account_request;
ALTER TABLE notifications DROP COLUMN account_outcome;
ALTER TABLE notifications DROP COLUMN account_name;
ALTER TABLE notifications DROP COLUMN account_id;
ALTER TABLE notifications DROP CONSTRAINT notifications_kind_check;
ALTER TABLE notifications ADD CONSTRAINT notifications_kind_check CHECK (kind IN ('needs_speaker_review', 'worker_unavailable', 'provider_unavailable', 'model_missing', 'job_failed', 'job_blocked', 'folder_request_decided'));
