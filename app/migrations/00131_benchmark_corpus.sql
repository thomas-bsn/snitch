-- +goose Up
-- A benchmark dataset becomes a corpus of annotated clips (#287, ADR 0057). The speaker-level items of
-- benchmark_dataset_items stay: they are derived from the clips' human-validated voices at creation.

-- Each clip has one role. A test clip is what a model is measured on; a reference clip only enrols the Persons
-- of the voice fingerprint stage, so a voice never serves as its own reference. verified is the administrator's
-- attestation that the whole transcript and the speakers of a test clip were reviewed by a human: the text
-- and the speaker boundaries come from the production pipeline, so an unreviewed clip would measure the
-- production models against themselves.
CREATE TABLE benchmark_dataset_clips (
    dataset_id bigint NOT NULL REFERENCES benchmark_datasets(id) ON DELETE CASCADE,
    clip_id bigint NOT NULL REFERENCES clips(id) ON DELETE CASCADE,
    role text NOT NULL CHECK (role IN ('reference','test')),
    verified boolean NOT NULL DEFAULT false,
    PRIMARY KEY (dataset_id, clip_id),
    CHECK (role = 'test' OR NOT verified)
);
CREATE INDEX benchmark_dataset_clips_clip ON benchmark_dataset_clips(clip_id);

-- The stages the corpus can measure, decided at creation from its content: fingerprint, transcription,
-- diarization, e2e. A stage listed here is not necessarily implemented yet; a run refuses the others.
ALTER TABLE benchmark_datasets ADD COLUMN stages text[] NOT NULL DEFAULT '{}';

-- +goose Down
ALTER TABLE benchmark_datasets DROP COLUMN stages;
DROP TABLE benchmark_dataset_clips;
