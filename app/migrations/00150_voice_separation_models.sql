-- +goose Up
-- A separation model can now be supported (#401, ADR 0065): the SpeechBrain family has a separation adapter.
-- Migration 00097 forbade it while none existed. The rule becomes: a supported separation model declares its
-- limits, the number of voices it separates and the sample rate it works at, read from its original documentation.
-- +goose StatementBegin
DO $$
DECLARE name text;
BEGIN
    -- The constraint of 00097 was created without a name: it is found by what it says.
    SELECT conname INTO STRICT name FROM pg_constraint
    WHERE conrelid = 'voice_model_catalog'::regclass AND contype = 'c'
      AND pg_get_constraintdef(oid) LIKE '%separation%' AND pg_get_constraintdef(oid) LIKE '%NOT supported%';
    EXECUTE format('ALTER TABLE voice_model_catalog DROP CONSTRAINT %I', name);
END $$;
-- +goose StatementEnd

-- 0 means not applicable (another type) or not known (a reference without an approved adapter).
ALTER TABLE voice_model_catalog ADD COLUMN separated_voices integer NOT NULL DEFAULT 0 CHECK (separated_voices >= 0);
ALTER TABLE voice_model_catalog ADD COLUMN sample_rate_hz integer NOT NULL DEFAULT 0 CHECK (sample_rate_hz >= 0);
ALTER TABLE voice_model_catalog ADD CONSTRAINT voice_model_catalog_separation_limits
    CHECK (kind <> 'separation' OR NOT supported OR (separated_voices >= 2 AND sample_rate_hz > 0));

-- The SepFormer adapter is a new engine name of the catalogue. It only exists on the on-demand SpeechBrain family.
ALTER TABLE voice_model_catalog DROP CONSTRAINT voice_model_catalog_engine_check;
ALTER TABLE voice_model_catalog ADD CONSTRAINT voice_model_catalog_engine_check
    CHECK (engine IN ('','pyannote','nemo','silero-jit','nemo-titanet','speechbrain-ecapa','wespeaker','speechbrain-sepformer'));

-- The two candidates of ADR 0065, pinned to the commit whose model card and hyperparams were read. Nothing is
-- installed by this: a model reaches a Worker on an explicit order only. A reference an owner had already added
-- for the same repository (unsupported until now) becomes that candidate.
INSERT INTO voice_model_catalog (id, kind, engine, repo, revision, label, license, gated, size_bytes, family, separated_voices, sample_rate_hz) VALUES
    ('speechbrain-sepformer-whamr16k', 'separation', 'speechbrain-sepformer', 'speechbrain/sepformer-whamr16k', '21a5b500c6f52fddc387c5d9e5fb13ffd6f039c5', 'SepFormer WHAMR! 16 kHz', 'apache-2.0', false, 318987701, 'speechbrain', 2, 16000),
    ('speechbrain-sepformer-wsj03mix', 'separation', 'speechbrain-sepformer', 'speechbrain/sepformer-wsj03mix', '21e78b44a378092c34fff32ba8e788a73584f474', 'SepFormer WSJ0-3Mix 8 kHz', 'apache-2.0', false, 319665231, 'speechbrain', 3, 8000)
ON CONFLICT (kind, repo) DO UPDATE SET engine = EXCLUDED.engine, revision = EXCLUDED.revision, license = EXCLUDED.license, gated = EXCLUDED.gated,
    size_bytes = EXCLUDED.size_bytes, family = EXCLUDED.family, separated_voices = EXCLUDED.separated_voices, sample_rate_hz = EXCLUDED.sample_rate_hz,
    supported = true, unsupported_reason = '', updated_at = now();

-- +goose Down
-- Nothing is deleted: every separation reference, the two candidates included, goes back to unsupported (the only
-- state 00097 allows) and stays listed. Replaying the migration makes the candidates supported again.
UPDATE voice_model_catalog SET supported = false, unsupported_reason = 'no approved adapter runs separation', engine = '', family = ''
    WHERE kind = 'separation' AND supported;
ALTER TABLE voice_model_catalog DROP CONSTRAINT voice_model_catalog_engine_check;
ALTER TABLE voice_model_catalog ADD CONSTRAINT voice_model_catalog_engine_check
    CHECK (engine IN ('','pyannote','nemo','silero-jit','nemo-titanet','speechbrain-ecapa','wespeaker'));
ALTER TABLE voice_model_catalog DROP CONSTRAINT voice_model_catalog_separation_limits;
ALTER TABLE voice_model_catalog DROP COLUMN sample_rate_hz;
ALTER TABLE voice_model_catalog DROP COLUMN separated_voices;
ALTER TABLE voice_model_catalog ADD CONSTRAINT voice_model_catalog_check CHECK (kind <> 'separation' OR NOT supported);
