-- +goose Up
-- The catalogue of voice models moves from the code to the database (ADR 0038). The ids of the three
-- historical references are kept: they are the keys of worker_voice_models and of voice_model_candidate,
-- which are therefore migrated without any change. There is deliberately no foreign key from those tables.
CREATE TABLE voice_model_catalog (
    id text PRIMARY KEY CHECK (length(id) BETWEEN 1 AND 64),
    kind text NOT NULL CHECK (kind IN ('diarization','fingerprint','separation')),
    -- The adapter allowed to run the reference; empty when no approved adapter exists.
    engine text NOT NULL DEFAULT '' CHECK (engine IN ('','pyannote','nemo')),
    repo text NOT NULL CHECK (length(repo) BETWEEN 1 AND 100),
    revision text NOT NULL CHECK (length(revision) BETWEEN 1 AND 64),
    label text NOT NULL DEFAULT '' CHECK (length(label) <= 100),
    license text NOT NULL DEFAULT '' CHECK (length(license) <= 100),
    gated boolean NOT NULL DEFAULT false,
    -- Known hardware needs; 0 means not known or not limited.
    max_speakers integer NOT NULL DEFAULT 0 CHECK (max_speakers >= 0),
    size_bytes bigint NOT NULL DEFAULT 0 CHECK (size_bytes >= 0),
    vram_bytes bigint NOT NULL DEFAULT 0 CHECK (vram_bytes >= 0),
    supports_num_speakers boolean NOT NULL DEFAULT false,
    -- A disabled reference cannot be chosen any more; its history stays readable.
    enabled boolean NOT NULL DEFAULT true,
    supported boolean NOT NULL DEFAULT true,
    unsupported_reason text NOT NULL DEFAULT '' CHECK (length(unsupported_reason) <= 200),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (kind, repo),
    -- No adapter exists for separation yet: such a reference can never be supported.
    CHECK (kind <> 'separation' OR NOT supported),
    CHECK (supported OR unsupported_reason <> ''),
    CHECK (NOT supported OR engine <> '')
);

INSERT INTO voice_model_catalog (id, kind, engine, repo, revision, license, gated, max_speakers, size_bytes, supports_num_speakers) VALUES
    ('pyannote-community-1', 'diarization', 'pyannote', 'pyannote/speaker-diarization-community-1', '3533c8cf8e369892e6b79ff1bf80f7b0286a54ee', 'cc-by-4.0', true, 0, 33695573, true),
    ('nvidia-sortformer-4spk-v2.1', 'diarization', 'nemo', 'nvidia/diar_streaming_sortformer_4spk-v2.1', 'cd03eee90fbec18297ac31b8c21546e596b7f71c', 'nvidia-open-model-license', false, 4, 471403401, false),
    ('pyannote-wespeaker-resnet34', 'fingerprint', 'pyannote', 'pyannote/wespeaker-voxceleb-resnet34-LM', '837717ddb9ff5507820346191109dc79c958d614', 'cc-by-4.0', false, 0, 26650477, false);

-- The active model of a type that is not stored in the versioned voice pipeline. The active diarization model
-- stays in the pipeline configuration, which the pinned jobs refer to; it is not copied here.
CREATE TABLE voice_active_model (
    kind text PRIMARY KEY CHECK (kind IN ('fingerprint','separation')),
    model_id text NOT NULL CHECK (length(model_id) BETWEEN 1 AND 64)
);

-- The fingerprint model that used to be a constant of the code.
INSERT INTO voice_active_model (kind, model_id) VALUES ('fingerprint', 'pyannote-wespeaker-resnet34');

-- +goose Down
DROP TABLE voice_active_model;
DROP TABLE voice_model_catalog;
