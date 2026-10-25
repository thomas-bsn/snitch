-- +goose Up
-- Versioned voice pipeline configuration (diarization model + fingerprint thresholds).
CREATE TABLE voice_pipeline_versions (
    version text PRIMARY KEY,
    config jsonb NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE voice_pipeline_active (
    id boolean PRIMARY KEY DEFAULT true CHECK (id),
    version text NOT NULL REFERENCES voice_pipeline_versions(version)
);
-- The reference model needs an access token and accepted conditions: nothing runs
-- until an administrator confirms access explicitly (access_confirmed).
INSERT INTO voice_pipeline_versions(version, config)
VALUES ('voice-v1', '{"provider":"pyannote","model":"pyannote/speaker-diarization-community-1","model_version":"community-1","access_confirmed":false,"min_fingerprint_seconds":1.0,"min_quality":0.5}');
INSERT INTO voice_pipeline_active(id, version) VALUES (true, 'voice-v1');

-- One row per voice detected in a clip by the current diarization of an audio stream.
CREATE TABLE local_speakers (
    id bigserial PRIMARY KEY,
    clip_id bigint NOT NULL REFERENCES clips(id) ON DELETE CASCADE,
    audio_stream_index integer NOT NULL CHECK (audio_stream_index >= 0),
    diarization_job_id bigint NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    label text NOT NULL CHECK (label <> '' AND length(label) <= 64),
    ordinal integer NOT NULL CHECK (ordinal >= 0),
    speaking_seconds double precision NOT NULL CHECK (speaking_seconds >= 0),
    turn_count integer NOT NULL CHECK (turn_count >= 0),
    UNIQUE (clip_id, audio_stream_index, label)
);
CREATE INDEX local_speakers_clip ON local_speakers(clip_id, audio_stream_index, ordinal);

CREATE TABLE speaker_turns (
    id bigserial PRIMARY KEY,
    local_speaker_id bigint NOT NULL REFERENCES local_speakers(id) ON DELETE CASCADE,
    clip_id bigint NOT NULL REFERENCES clips(id) ON DELETE CASCADE,
    turn_index integer NOT NULL CHECK (turn_index >= 0),
    start_seconds double precision NOT NULL CHECK (start_seconds >= 0),
    end_seconds double precision NOT NULL,
    CHECK (end_seconds >= start_seconds),
    UNIQUE (local_speaker_id, turn_index)
);
CREATE INDEX speaker_turns_clip ON speaker_turns(clip_id, start_seconds);

-- Fingerprints keep their model and version: vectors of different models or
-- dimensions must never be compared. usable=false rows are kept for display but
-- must never serve as references or as a propagation base.
CREATE TABLE voice_fingerprints (
    id bigserial PRIMARY KEY,
    local_speaker_id bigint NOT NULL REFERENCES local_speakers(id) ON DELETE CASCADE,
    clip_id bigint NOT NULL REFERENCES clips(id) ON DELETE CASCADE,
    fingerprint_job_id bigint NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    scope text NOT NULL CHECK (scope IN ('speaker','turn')),
    turn_id bigint REFERENCES speaker_turns(id) ON DELETE CASCADE,
    start_seconds double precision NOT NULL CHECK (start_seconds >= 0),
    end_seconds double precision NOT NULL,
    duration_seconds double precision NOT NULL CHECK (duration_seconds >= 0),
    quality double precision NOT NULL CHECK (quality BETWEEN 0 AND 1),
    usable boolean NOT NULL,
    unusable_reason text NOT NULL DEFAULT '' CHECK (unusable_reason IN ('','too_short','low_quality')),
    model text NOT NULL CHECK (model <> ''),
    model_version text NOT NULL CHECK (model_version <> ''),
    dimensions integer NOT NULL CHECK (dimensions BETWEEN 1 AND 2000),
    embedding vector NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CHECK (end_seconds >= start_seconds),
    CHECK (vector_dims(embedding) = dimensions),
    CHECK (usable = (unusable_reason = ''))
);
CREATE INDEX voice_fingerprints_clip ON voice_fingerprints(clip_id, local_speaker_id);
CREATE INDEX voice_fingerprints_model ON voice_fingerprints(model, model_version, dimensions) WHERE usable;

-- +goose Down
DROP TABLE voice_fingerprints;
DROP TABLE speaker_turns;
DROP TABLE local_speakers;
DROP TABLE voice_pipeline_active;
DROP TABLE voice_pipeline_versions;
