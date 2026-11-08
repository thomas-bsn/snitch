-- +goose Up
-- A finished clip stays under "Awaiting review" in the Task Queue while a voice awaits a human
-- decision. Any decision, disposition or
-- re-matching that changes the voices tells the 'admin:queue' topic to send a fresh snapshot.
CREATE TRIGGER speaker_attributions_admin_queue_event AFTER INSERT OR UPDATE OR DELETE ON speaker_attributions FOR EACH STATEMENT EXECUTE FUNCTION admin_queue_event();
CREATE TRIGGER local_speaker_dispositions_admin_queue_event AFTER INSERT OR UPDATE OR DELETE ON local_speaker_dispositions FOR EACH STATEMENT EXECUTE FUNCTION admin_queue_event();
CREATE TRIGGER local_speakers_admin_queue_event AFTER INSERT OR UPDATE OR DELETE ON local_speakers FOR EACH STATEMENT EXECUTE FUNCTION admin_queue_event();
-- +goose Down
DROP TRIGGER local_speakers_admin_queue_event ON local_speakers;
DROP TRIGGER local_speaker_dispositions_admin_queue_event ON local_speaker_dispositions;
DROP TRIGGER speaker_attributions_admin_queue_event ON speaker_attributions;
