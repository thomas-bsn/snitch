-- +goose Up
-- Relaunching the diarization of a clip with human validations. The Local speakers a
-- human validated are not deleted by the new diarization: they become 'archived'
-- (their validated Attribution, their Voice samples and their speaker fingerprints stay,
-- linked to the audio), while the new result adds fresh 'diarized' speakers. An archived
-- speaker keeps no turn, so it takes part neither in the transcript nor in matching of
-- unknown voices.
ALTER TABLE local_speakers DROP CONSTRAINT local_speakers_kind_check;
ALTER TABLE local_speakers ADD CONSTRAINT local_speakers_kind_check CHECK (kind IN ('diarized','microphone','archived'));
ALTER TABLE local_speakers ADD COLUMN archived_at timestamptz;
ALTER TABLE local_speakers ADD CONSTRAINT local_speakers_archived_at CHECK ((kind = 'archived') = (archived_at IS NOT NULL));
-- A label is unique among the live speakers of a stream only.
ALTER TABLE local_speakers DROP CONSTRAINT local_speakers_clip_id_audio_stream_index_label_key;
CREATE UNIQUE INDEX local_speakers_live_label ON local_speakers(clip_id, audio_stream_index, label) WHERE kind <> 'archived';

-- +goose Down
DROP INDEX local_speakers_live_label;
DELETE FROM local_speakers WHERE kind = 'archived';
ALTER TABLE local_speakers ADD CONSTRAINT local_speakers_clip_id_audio_stream_index_label_key UNIQUE (clip_id, audio_stream_index, label);
ALTER TABLE local_speakers DROP CONSTRAINT local_speakers_archived_at;
ALTER TABLE local_speakers DROP COLUMN archived_at;
ALTER TABLE local_speakers DROP CONSTRAINT local_speakers_kind_check;
ALTER TABLE local_speakers ADD CONSTRAINT local_speakers_kind_check CHECK (kind IN ('diarized','microphone'));
