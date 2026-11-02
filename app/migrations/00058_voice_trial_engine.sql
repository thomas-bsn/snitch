-- +goose Up
-- The engine of a trial is the one of the catalogue model it runs (pyannote, nemo), kept with the pinned
-- repository and revision so the Worker is never told another engine.
ALTER TABLE voice_diarization_trials ADD COLUMN model_engine text NOT NULL DEFAULT 'pyannote';

-- +goose Down
ALTER TABLE voice_diarization_trials DROP COLUMN model_engine;
