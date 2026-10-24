-- +goose Up
CREATE TABLE source_audio_track_rules (
    source_id bigint PRIMARY KEY REFERENCES clip_sources(id) ON DELETE CASCADE,
    rule jsonb NOT NULL,
    version bigint NOT NULL DEFAULT 1,
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE clip_audio_track_overrides (
    clip_id bigint PRIMARY KEY REFERENCES clips(id) ON DELETE CASCADE,
    audio_stream_index integer NOT NULL CHECK (audio_stream_index >= 0),
    version bigint NOT NULL DEFAULT 1,
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE clip_audio_track_overrides;
DROP TABLE source_audio_track_rules;
