-- +goose Up
-- The offline speech detector of the audio diagnostics (ADR 0048) is a catalogue model like the voice models:
-- it is installed only by an explicit order and a Worker without it never receives a diagnostic job.
ALTER TABLE voice_model_catalog DROP CONSTRAINT voice_model_catalog_kind_check;
ALTER TABLE voice_model_catalog ADD CONSTRAINT voice_model_catalog_kind_check CHECK (kind IN ('diarization','fingerprint','separation','vad'));
ALTER TABLE voice_model_catalog DROP CONSTRAINT voice_model_catalog_engine_check;
ALTER TABLE voice_model_catalog ADD CONSTRAINT voice_model_catalog_engine_check CHECK (engine IN ('','pyannote','nemo','silero-jit'));
ALTER TABLE worker_model_orders DROP CONSTRAINT worker_model_orders_kind_check;
ALTER TABLE worker_model_orders ADD CONSTRAINT worker_model_orders_kind_check CHECK (kind IN ('diarization','fingerprint','separation','vad'));
ALTER TABLE worker_model_replacements DROP CONSTRAINT worker_model_replacements_kind_check;
ALTER TABLE worker_model_replacements ADD CONSTRAINT worker_model_replacements_kind_check CHECK (kind IN ('diarization','fingerprint','separation','vad'));
INSERT INTO voice_model_catalog (id, kind, engine, repo, revision, label, license, gated, size_bytes) VALUES
    ('silero-vad-v6.2.1', 'vad', 'silero-jit', 'snakers4/silero-vad', '7e30209a3e901f9842f81b225f3e93d8199902b1', 'Silero VAD 6.2.1 (CPU)', 'mit', false, 2272526);

-- +goose Down
DELETE FROM worker_voice_models WHERE model_id IN (SELECT id FROM voice_model_catalog WHERE kind='vad');
DELETE FROM worker_model_orders WHERE kind='vad';
DELETE FROM worker_model_replacements WHERE kind='vad';
DELETE FROM voice_model_catalog WHERE kind='vad';
ALTER TABLE worker_model_replacements DROP CONSTRAINT worker_model_replacements_kind_check;
ALTER TABLE worker_model_replacements ADD CONSTRAINT worker_model_replacements_kind_check CHECK (kind IN ('diarization','fingerprint','separation'));
ALTER TABLE worker_model_orders DROP CONSTRAINT worker_model_orders_kind_check;
ALTER TABLE worker_model_orders ADD CONSTRAINT worker_model_orders_kind_check CHECK (kind IN ('diarization','fingerprint','separation'));
ALTER TABLE voice_model_catalog DROP CONSTRAINT voice_model_catalog_engine_check;
ALTER TABLE voice_model_catalog ADD CONSTRAINT voice_model_catalog_engine_check CHECK (engine IN ('','pyannote','nemo'));
ALTER TABLE voice_model_catalog DROP CONSTRAINT voice_model_catalog_kind_check;
ALTER TABLE voice_model_catalog ADD CONSTRAINT voice_model_catalog_kind_check CHECK (kind IN ('diarization','fingerprint','separation'));
