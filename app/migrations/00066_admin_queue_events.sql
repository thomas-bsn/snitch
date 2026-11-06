-- +goose Up
-- The Task Queue follows every analysis run, step and job: any change on them tells the
-- 'admin:queue' topic to send a fresh snapshot. Statement-level triggers keep the outbox small.
-- +goose StatementBegin
CREATE FUNCTION admin_queue_event() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 INSERT INTO events(topic, kind) VALUES ('admin:queue', 'queue');
 RETURN NULL;
END $$;
-- +goose StatementEnd
CREATE TRIGGER jobs_admin_queue_event AFTER INSERT OR UPDATE OR DELETE ON jobs FOR EACH STATEMENT EXECUTE FUNCTION admin_queue_event();
CREATE TRIGGER clip_analysis_runs_admin_queue_event AFTER INSERT OR UPDATE OR DELETE ON clip_analysis_runs FOR EACH STATEMENT EXECUTE FUNCTION admin_queue_event();
CREATE TRIGGER clip_analysis_steps_admin_queue_event AFTER INSERT OR UPDATE OR DELETE ON clip_analysis_steps FOR EACH STATEMENT EXECUTE FUNCTION admin_queue_event();
-- +goose Down
DROP TRIGGER clip_analysis_steps_admin_queue_event ON clip_analysis_steps;
DROP TRIGGER clip_analysis_runs_admin_queue_event ON clip_analysis_runs;
DROP TRIGGER jobs_admin_queue_event ON jobs;
DROP FUNCTION admin_queue_event();
