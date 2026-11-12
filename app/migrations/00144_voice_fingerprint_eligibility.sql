-- +goose Up
-- Eligibility of a fingerprint decided by Snitch (ADR 0064, #390). The Worker reports measures (00142); Snitch
-- applies a versioned rule and can requalify every stored fingerprint in the database, without any job.
--
-- Two uses: a comparable fingerprint is matched on its own against a Reference (the existing column `usable`
-- keeps that meaning, so the partial indexes `WHERE usable` stay right); a contributive one may enter a
-- Reference once validated by a human. Comparable implies contributive: the contributive duration is lower.
--
-- Append-only history of the rule: the active rule is the highest version. Version 1 is the provisional rule of
-- Snitch (no author), meant to be replaced by the administrator after the figures of the preview (#394).
-- Authors are snapshots, without foreign key: accounts may disappear.
CREATE TABLE voice_eligibility_rules (
    version integer PRIMARY KEY CHECK (version >= 1),
    comparable_min_seconds double precision NOT NULL CHECK (comparable_min_seconds BETWEEN 0.5 AND 60),
    contributive_min_seconds double precision NOT NULL CHECK (contributive_min_seconds BETWEEN 0.5 AND 60),
    max_clipped_ratio double precision NOT NULL CHECK (max_clipped_ratio BETWEEN 0 AND 1),
    author_id bigint,
    author_username text,
    created_at timestamptz NOT NULL DEFAULT now(),
    CHECK (contributive_min_seconds <= comparable_min_seconds)
);
INSERT INTO voice_eligibility_rules(version, comparable_min_seconds, contributive_min_seconds, max_clipped_ratio)
VALUES (1, 2.0, 1.0, 0.01);

-- Net speech of a local speaker (p_turn_id null) or of one of its turns, from the stored turns: its turns minus
-- the turns of the other local speakers of the same diarization, keeping the net intervals of at least 0.5 s
-- (the Worker's definition, measures version 1). No row when the scope has no stored turn (a validated speaker
-- archived by a later diarization keeps its fingerprint but loses its turns).
-- +goose StatementBegin
CREATE FUNCTION voice_turns_net_speech(p_local_speaker_id bigint, p_turn_id bigint)
RETURNS TABLE (net_speech_seconds double precision, overlap_ratio double precision, overlapped boolean)
LANGUAGE sql STABLE AS $$
    WITH own AS (
        SELECT range_agg(numrange(st.start_seconds::numeric, st.end_seconds::numeric)) AS spans
        FROM speaker_turns st
        WHERE st.local_speaker_id = p_local_speaker_id AND (p_turn_id IS NULL OR st.id = p_turn_id)
    ), others AS (
        SELECT COALESCE(range_agg(numrange(st.start_seconds::numeric, st.end_seconds::numeric)), '{}'::nummultirange) AS spans
        FROM local_speakers me
        JOIN local_speakers ls ON ls.diarization_job_id = me.diarization_job_id AND ls.id <> me.id
        JOIN speaker_turns st ON st.local_speaker_id = ls.id
        WHERE me.id = p_local_speaker_id
    ), geometry AS (
        SELECT own.spans AS own, own.spans - others.spans AS net FROM own, others WHERE NOT isempty(own.spans)
    )
    SELECT kept.seconds::double precision,
           LEAST(1, GREATEST(0, round(1 - COALESCE(free.seconds, 0) / whole.seconds, 4)))::double precision,
           kept.seconds = 0
    FROM geometry g,
         LATERAL (SELECT COALESCE(sum(upper(p) - lower(p)) FILTER (WHERE upper(p) - lower(p) >= 0.5), 0) AS seconds FROM unnest(g.net) p) kept,
         LATERAL (SELECT sum(upper(p) - lower(p)) AS seconds FROM unnest(g.net) p) free,
         LATERAL (SELECT sum(upper(p) - lower(p)) AS seconds FROM unnest(g.own) p) whole
$$;
-- +goose StatementEnd

--   contributive               may enter a Reference once validated (comparable implies contributive)
--   eligibility_rule_version   rule that decided `usable`/`contributive`; null: decided when the fingerprint was
--                              stored, before the rule (existing rows until a requalification) or for benchmark
--                              and pending transition fingerprints, never requalified
--   net_speech_source          origin of net_speech_seconds/overlap_ratio/overlapped: 'worker' (measures),
--                              'turns' (stored turns), 'estimated' (no measures, no turns: the fingerprint's duration)
ALTER TABLE voice_fingerprints
    ADD COLUMN contributive boolean,
    ADD COLUMN eligibility_rule_version integer,
    ADD COLUMN net_speech_source text CHECK (net_speech_source IN ('worker', 'turns', 'estimated'));
UPDATE voice_fingerprints SET net_speech_source = 'worker' WHERE measures_version IS NOT NULL;
-- Existing production fingerprints keep their decision until the administrator requalifies them; their net speech
-- is derived now, while the turns are still stored, never above the fingerprint's duration (the speech embedded).
UPDATE voice_fingerprints f
SET net_speech_seconds = LEAST(g.net_speech_seconds, f.duration_seconds), overlap_ratio = g.overlap_ratio, overlapped = g.overlapped,
    net_speech_source = CASE WHEN g.net_speech_seconds IS NULL THEN 'estimated' ELSE 'turns' END
FROM voice_fingerprints x
LEFT JOIN LATERAL voice_turns_net_speech(x.local_speaker_id, x.turn_id) g ON x.scope = 'speaker' OR x.turn_id IS NOT NULL
WHERE x.id = f.id AND f.net_speech_source IS NULL AND f.measures_version IS NULL AND NOT f.benchmark;
-- The new reasons are allowed before an overlapped fingerprint is marked with one of them.
ALTER TABLE voice_fingerprints DROP CONSTRAINT voice_fingerprints_unusable_reason_check;
ALTER TABLE voice_fingerprints
    ADD CONSTRAINT voice_fingerprints_unusable_reason_check CHECK (unusable_reason IN ('', 'too_short', 'low_quality', 'overlap_only', 'clipped'));
-- An overlapped fingerprint is never eligible, whatever the rule.
UPDATE voice_fingerprints SET usable = false, unusable_reason = 'overlap_only' WHERE overlapped;
UPDATE voice_fingerprints SET contributive = usable;
ALTER TABLE voice_fingerprints ALTER COLUMN contributive SET NOT NULL;
ALTER TABLE voice_fingerprints
    ADD CONSTRAINT voice_fingerprints_comparable_contributes CHECK (NOT usable OR contributive),
    ADD CONSTRAINT voice_fingerprints_contributive_reason CHECK (NOT contributive OR unusable_reason IN ('', 'too_short')),
    ADD CONSTRAINT voice_fingerprints_overlapped_never_contributes CHECK (NOT (contributive AND overlapped IS TRUE));

-- +goose Down
UPDATE voice_fingerprints SET unusable_reason = 'too_short' WHERE unusable_reason IN ('overlap_only', 'clipped');
ALTER TABLE voice_fingerprints
    DROP CONSTRAINT voice_fingerprints_overlapped_never_contributes,
    DROP CONSTRAINT voice_fingerprints_contributive_reason,
    DROP CONSTRAINT voice_fingerprints_comparable_contributes,
    DROP CONSTRAINT voice_fingerprints_unusable_reason_check;
ALTER TABLE voice_fingerprints
    ADD CONSTRAINT voice_fingerprints_unusable_reason_check CHECK (unusable_reason IN ('', 'too_short', 'low_quality'));
UPDATE voice_fingerprints SET net_speech_seconds = NULL, overlap_ratio = NULL, overlapped = NULL WHERE measures_version IS NULL;
ALTER TABLE voice_fingerprints
    DROP COLUMN net_speech_source,
    DROP COLUMN eligibility_rule_version,
    DROP COLUMN contributive;
DROP FUNCTION voice_turns_net_speech(bigint, bigint);
DROP TABLE voice_eligibility_rules;
