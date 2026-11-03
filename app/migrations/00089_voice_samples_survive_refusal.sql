-- +goose Up
-- A refusal of an identification (ADR 0031) removes the validated attribution but only
-- retires its reference sample, like a correction: the history (attribution_events
-- .old_sample_id) keeps pointing to it. The sample therefore outlives its attribution, whose
-- deletion now clears the reference instead of deleting the sample. An active sample (a
-- reference) still always has its attribution.
ALTER TABLE voice_samples ALTER COLUMN attribution_id DROP NOT NULL;
ALTER TABLE voice_samples DROP CONSTRAINT voice_samples_attribution_id_fkey;
ALTER TABLE voice_samples ADD CONSTRAINT voice_samples_attribution_id_fkey
    FOREIGN KEY (attribution_id) REFERENCES speaker_attributions(id) ON DELETE SET NULL;
ALTER TABLE voice_samples ADD CONSTRAINT voice_samples_active_has_attribution
    CHECK (retired_at IS NOT NULL OR attribution_id IS NOT NULL);

-- +goose Down
DELETE FROM voice_samples WHERE attribution_id IS NULL;
ALTER TABLE voice_samples DROP CONSTRAINT voice_samples_active_has_attribution;
ALTER TABLE voice_samples DROP CONSTRAINT voice_samples_attribution_id_fkey;
ALTER TABLE voice_samples ADD CONSTRAINT voice_samples_attribution_id_fkey
    FOREIGN KEY (attribution_id) REFERENCES speaker_attributions(id) ON DELETE CASCADE;
ALTER TABLE voice_samples ALTER COLUMN attribution_id SET NOT NULL;
