-- +goose Up
-- Dedicated microphone track (phase 5). An administrator designates, per
-- source, which detected audio track is a microphone and which Person owns it; the
-- track then receives that Person directly, without diarization. A clip can replace
-- the choice. Rules reference the tracks really detected (index and/or exact title);
-- no title is assumed.

-- A microphone local speaker is not produced by a diarization job.
ALTER TABLE local_speakers ADD COLUMN kind text NOT NULL DEFAULT 'diarized' CHECK (kind IN ('diarized','microphone'));
ALTER TABLE local_speakers ALTER COLUMN diarization_job_id DROP NOT NULL;
ALTER TABLE local_speakers ADD CONSTRAINT local_speakers_kind_job CHECK ((kind = 'microphone') = (diarization_job_id IS NULL));
-- At most one microphone speaker per clip.
CREATE UNIQUE INDEX local_speakers_microphone ON local_speakers(clip_id) WHERE kind = 'microphone';

-- Source rule: the track is matched by global stream index and/or exact title (both
-- must match when both are given). The Person is RESTRICTed: deleting a Person must
-- first remove the rules that name it.
CREATE TABLE source_microphone_tracks (
    source_id bigint PRIMARY KEY REFERENCES clip_sources(id) ON DELETE CASCADE,
    audio_stream_index integer CHECK (audio_stream_index IS NULL OR audio_stream_index >= 0),
    title text CHECK (title IS NULL OR (btrim(title) <> '' AND length(title) <= 200)),
    person_id bigint NOT NULL REFERENCES persons(id),
    version bigint NOT NULL DEFAULT 1,
    updated_by bigint REFERENCES users(id) ON DELETE SET NULL,
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK (audio_stream_index IS NOT NULL OR title IS NOT NULL)
);

-- Clip replacement: either another track and owner, or "no microphone track here".
CREATE TABLE clip_microphone_overrides (
    clip_id bigint PRIMARY KEY REFERENCES clips(id) ON DELETE CASCADE,
    disabled boolean NOT NULL DEFAULT false,
    audio_stream_index integer CHECK (audio_stream_index IS NULL OR audio_stream_index >= 0),
    person_id bigint REFERENCES persons(id),
    version bigint NOT NULL DEFAULT 1,
    updated_by bigint REFERENCES users(id) ON DELETE SET NULL,
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK (disabled = (audio_stream_index IS NULL)),
    CHECK ((audio_stream_index IS NULL) = (person_id IS NULL))
);

-- Append-only history of every designation change. Rule snapshots carry the Person
-- name at the time and no foreign key, so the trail survives later deletions. Rows
-- disappear only through the cascade from their clip.
CREATE TABLE microphone_track_events (
    id bigserial PRIMARY KEY,
    scope text NOT NULL CHECK (scope IN ('source','clip')),
    source_id bigint,
    clip_id bigint REFERENCES clips(id) ON DELETE CASCADE,
    action text NOT NULL CHECK (action IN ('set','remove','disable')),
    old_rule jsonb,
    new_rule jsonb,
    author_id bigint,
    author_username text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    CHECK ((scope = 'source' AND source_id IS NOT NULL) OR (scope = 'clip' AND clip_id IS NOT NULL))
);
CREATE INDEX microphone_track_events_clip ON microphone_track_events(clip_id, id) WHERE clip_id IS NOT NULL;
CREATE INDEX microphone_track_events_source ON microphone_track_events(source_id, id) WHERE scope = 'source';
CREATE TRIGGER microphone_track_events_immutable BEFORE UPDATE OR DELETE ON microphone_track_events
    FOR EACH ROW EXECUTE FUNCTION attribution_events_immutable();

-- +goose Down
DROP TABLE microphone_track_events;
DROP TABLE clip_microphone_overrides;
DROP TABLE source_microphone_tracks;
ALTER TABLE local_speakers DISABLE TRIGGER local_speakers_keep_validated;
DELETE FROM local_speakers WHERE kind = 'microphone';
ALTER TABLE local_speakers ENABLE TRIGGER local_speakers_keep_validated;
DROP INDEX local_speakers_microphone;
ALTER TABLE local_speakers DROP CONSTRAINT local_speakers_kind_job;
ALTER TABLE local_speakers ALTER COLUMN diarization_job_id SET NOT NULL;
ALTER TABLE local_speakers DROP COLUMN kind;
