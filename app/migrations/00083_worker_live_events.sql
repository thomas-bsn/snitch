-- +goose Up
-- Live Worker pages: a change that an administrator can see writes one event on the list topic
-- (admin:workers) and one on the topic of that Worker (worker:<id>). Streams answer with a fresh
-- snapshot, so the events carry no data. Pulses only count when they change what is shown: the
-- connection flips from Offline to Online (its last pulse was older than the connected window of
-- 30 s); the Offline flip happens without any write and is published by the server (see
-- workers.Service.PublishDisconnections).
-- +goose StatementBegin
CREATE FUNCTION worker_event() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE wid text := CASE WHEN TG_OP = 'DELETE' THEN OLD.id ELSE NEW.id END;
BEGIN
 INSERT INTO events(topic, kind) VALUES ('admin:workers', 'workers'), ('worker:' || wid, 'workers');
 RETURN NULL;
END $$;
-- +goose StatementEnd
CREATE TRIGGER workers_live_insert AFTER INSERT ON workers FOR EACH ROW EXECUTE FUNCTION worker_event();
CREATE TRIGGER workers_live_delete AFTER DELETE ON workers FOR EACH ROW EXECUTE FUNCTION worker_event();
CREATE TRIGGER workers_live_update AFTER UPDATE ON workers FOR EACH ROW
 WHEN (
  (OLD.name, OLD.max_concurrency, OLD.reported_concurrency, OLD.revoked_at, OLD.revoke_reason, OLD.revocation_acknowledged_at, OLD.models, OLD.transcription_model, OLD.supported_steps, OLD.machine, OLD.remote_address)
  IS DISTINCT FROM
  (NEW.name, NEW.max_concurrency, NEW.reported_concurrency, NEW.revoked_at, NEW.revoke_reason, NEW.revocation_acknowledged_at, NEW.models, NEW.transcription_model, NEW.supported_steps, NEW.machine, NEW.remote_address)
  OR (OLD.last_seen_at IS NULL OR OLD.last_seen_at < now() - interval '30 seconds')
 )
 EXECUTE FUNCTION worker_event();
-- Transcription configuration and installation progress are shown on the same pages.
-- +goose StatementBegin
CREATE FUNCTION worker_child_event() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 INSERT INTO events(topic, kind) VALUES ('admin:workers', 'workers'), ('worker:' || NEW.worker_id, 'workers');
 RETURN NULL;
END $$;
-- +goose StatementEnd
CREATE TRIGGER worker_transcription_config_live AFTER INSERT OR UPDATE ON worker_transcription_config FOR EACH ROW EXECUTE FUNCTION worker_child_event();
CREATE TRIGGER worker_install_events_live AFTER INSERT ON worker_install_events FOR EACH ROW EXECUTE FUNCTION worker_child_event();
-- The recent tasks of a Worker's page: only what a Worker did (job_events.actor = 'worker').
-- +goose StatementBegin
CREATE FUNCTION worker_task_event() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 INSERT INTO events(topic, kind) VALUES ('worker:' || NEW.actor_id, 'workers');
 RETURN NULL;
END $$;
-- +goose StatementEnd
CREATE TRIGGER job_events_worker_live AFTER INSERT ON job_events FOR EACH ROW WHEN (NEW.actor = 'worker') EXECUTE FUNCTION worker_task_event();
-- +goose Down
DROP TRIGGER job_events_worker_live ON job_events;
DROP FUNCTION worker_task_event();
DROP TRIGGER worker_install_events_live ON worker_install_events;
DROP TRIGGER worker_transcription_config_live ON worker_transcription_config;
DROP FUNCTION worker_child_event();
DROP TRIGGER workers_live_update ON workers;
DROP TRIGGER workers_live_delete ON workers;
DROP TRIGGER workers_live_insert ON workers;
DROP FUNCTION worker_event();
