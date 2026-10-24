-- +goose Up
CREATE TABLE transcription_pipeline_versions (
    version text PRIMARY KEY,
    config jsonb NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE transcription_pipeline_active (
    id boolean PRIMARY KEY DEFAULT true CHECK (id),
    version text NOT NULL REFERENCES transcription_pipeline_versions(version)
);

-- Version 1 keeps the existing worker configuration as the default profile.
INSERT INTO transcription_pipeline_versions(version, config)
VALUES ('pipeline-v1', '{"provider":"openai-compatible","model":"default","language":"fr","preparation":"wav-16khz-mono-v1","vad":{"mode":"off","version":"vad-v1"}}');
INSERT INTO transcription_pipeline_active(id, version) VALUES (true, 'pipeline-v1');

-- Keep jobs queued by version 3 consumable after the schema upgrade.
UPDATE jobs
SET input_version = 't4-a'
    || substring(input_version from 'transcription-v3-audio-([0-9]+)-rule-')
    || '-r'
    || substring(input_version from '-rule-([0-9]+)-vocab-')
    || '-v'
    || substring(input_version from '-vocab-([0-9a-f]+)$')
    || '-ppipeline-v1'
WHERE step LIKE 'transcription.audio.%'
  AND input_version ~ '^transcription-v3-audio-[0-9]+-rule-[0-9]+-vocab-[0-9a-f]+$';

-- +goose Down
DROP TABLE transcription_pipeline_active;
DROP TABLE transcription_pipeline_versions;
