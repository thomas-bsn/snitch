-- +goose Up
-- Quality measures of a fingerprint (ADR 0064, #389). The Worker computes the vector on the net speech of a local
-- speaker and reports what it measured; Snitch keeps the measures to decide eligibility itself (#390). Every column
-- is null for a fingerprint stored before the measures, or sent by an older Worker: unknown, never zero.
--   measures_version   version of the Worker's definitions (1: net speech intervals of at least 0.5 s)
--   net_speech_seconds net speech in the vector; 0 for a fingerprint of overlapped turns
--   overlap_ratio      share of the speech of the scope during which another local speaker of the track speaks
--   overlapped         no net interval was long enough: the vector is made of whole turns (never a reference)
--   clipped_ratio      share of clipped 16-bit samples of the audio embedded (null: audio not measured)
--   mean_level_dbfs    RMS level of the audio embedded, in dBFS, floored at -120 (null: audio not measured)
--   coherence          mean pairwise cosine of the speaker's turn vectors (null: fewer than two turns, or a turn)
ALTER TABLE voice_fingerprints
    ADD COLUMN measures_version integer CHECK (measures_version >= 1),
    ADD COLUMN net_speech_seconds double precision CHECK (net_speech_seconds >= 0),
    ADD COLUMN overlap_ratio double precision CHECK (overlap_ratio BETWEEN 0 AND 1),
    ADD COLUMN overlapped boolean,
    ADD COLUMN clipped_ratio double precision CHECK (clipped_ratio BETWEEN 0 AND 1),
    ADD COLUMN mean_level_dbfs double precision CHECK (mean_level_dbfs BETWEEN -120 AND 0),
    ADD COLUMN coherence double precision CHECK (coherence BETWEEN -1 AND 1);

-- +goose Down
ALTER TABLE voice_fingerprints
    DROP COLUMN coherence,
    DROP COLUMN mean_level_dbfs,
    DROP COLUMN clipped_ratio,
    DROP COLUMN overlapped,
    DROP COLUMN overlap_ratio,
    DROP COLUMN net_speech_seconds,
    DROP COLUMN measures_version;
