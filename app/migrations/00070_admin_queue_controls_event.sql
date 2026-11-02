-- +goose Up
-- The Task Queue summary shows the pause and the concurrency ceiling: changing them tells the
-- 'admin:queue' topic to send a fresh snapshot, like any change on the jobs.
CREATE TRIGGER job_control_admin_queue_event AFTER INSERT OR UPDATE OR DELETE ON job_control FOR EACH STATEMENT EXECUTE FUNCTION admin_queue_event();
-- +goose Down
DROP TRIGGER job_control_admin_queue_event ON job_control;
