-- +goose Up
-- Generic outbox of change events. A row only says "something changed on this topic": streams
-- read it to know when to send a fresh snapshot, so a lost event never leaves a stale view.
-- Events are written by triggers in the transaction of the change itself.
CREATE TABLE events (
 id bigserial PRIMARY KEY,
 topic text NOT NULL,
 kind text NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX events_topic ON events (topic, id DESC);
-- +goose StatementBegin
CREATE FUNCTION clip_analysis_step_event() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 INSERT INTO events(topic, kind) SELECT 'clip:' || clip_id, 'analysis' FROM clip_analysis_runs WHERE id = NEW.run_id;
 RETURN NULL;
END $$;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE FUNCTION clip_analysis_run_event() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 INSERT INTO events(topic, kind) VALUES ('clip:' || NEW.clip_id, 'analysis');
 RETURN NULL;
END $$;
-- +goose StatementEnd
CREATE TRIGGER clip_analysis_steps_event AFTER INSERT OR UPDATE ON clip_analysis_steps FOR EACH ROW EXECUTE FUNCTION clip_analysis_step_event();
CREATE TRIGGER clip_analysis_runs_event AFTER INSERT OR UPDATE ON clip_analysis_runs FOR EACH ROW EXECUTE FUNCTION clip_analysis_run_event();
-- +goose Down
DROP TRIGGER clip_analysis_runs_event ON clip_analysis_runs;
DROP TRIGGER clip_analysis_steps_event ON clip_analysis_steps;
DROP FUNCTION clip_analysis_run_event();
DROP FUNCTION clip_analysis_step_event();
DROP TABLE events;
