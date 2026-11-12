-- +goose Up
-- Rule that gives the words of a transcript to the Local speakers (ADR 0065, #398). 'word' gives each word to the
-- voice whose turns cover it most, so a segment reads as parts; 'segment' is the rule of before, the whole segment
-- to the voice that speaks most over it. The rule is applied when a transcript is read: changing it rewrites nothing.
--
-- Append-only history: the active rule is the highest version. Version 1 is Snitch's default (no author).
-- Authors are snapshots, without foreign key: accounts may disappear.
CREATE TABLE speaker_attribution_rules (
    version integer PRIMARY KEY CHECK (version >= 1),
    rule text NOT NULL CHECK (rule IN ('segment', 'word')),
    author_id bigint,
    author_username text,
    created_at timestamptz NOT NULL DEFAULT now()
);
INSERT INTO speaker_attribution_rules(version, rule) VALUES (1, 'word');

-- The voice sharing the most time with [p_start,p_end] among turns given as parallel arrays, grouped by voice in
-- tie-break order. Among equals the previous voice wins when given, else the first one. NULL when none overlaps.
-- Twin of dominant() in app/internal/segmentparts.
-- +goose StatementBegin
CREATE FUNCTION dominant_turn_speaker(p_ids bigint[], p_starts double precision[], p_ends double precision[],
    p_start double precision, p_end double precision, p_previous bigint) RETURNS bigint
LANGUAGE plpgsql IMMUTABLE AS $$
DECLARE
    best bigint;
    best_overlap double precision := 0;
    previous_overlap double precision := 0;
    cur bigint;
    cur_overlap double precision := 0;
    shared double precision;
    k int;
    n int := COALESCE(array_length(p_ids, 1), 0);
BEGIN
    IF p_start IS NULL OR p_end IS NULL THEN
        RETURN NULL;
    END IF;
    FOR k IN 1..n + 1 LOOP
        IF k > n OR cur IS DISTINCT FROM p_ids[k] THEN
            IF cur IS NOT NULL AND cur_overlap > 0 THEN
                IF best IS NULL OR cur_overlap > best_overlap THEN
                    best := cur;
                    best_overlap := cur_overlap;
                END IF;
                IF cur = p_previous THEN
                    previous_overlap := cur_overlap;
                END IF;
            END IF;
            EXIT WHEN k > n;
            cur := p_ids[k];
            cur_overlap := 0;
        END IF;
        shared := LEAST(p_end, p_ends[k]) - GREATEST(p_start, p_starts[k]);
        IF shared > 0 THEN
            cur_overlap := cur_overlap + shared;
        END IF;
    END LOOP;
    IF best IS NOT NULL AND p_previous IS NOT NULL AND previous_overlap = best_overlap THEN
        RETURN p_previous;
    END IF;
    RETURN best;
END $$;
-- +goose StatementEnd

-- The Local speakers of the parts of one visible segment under the word rule, for the Person filter of the search.
-- Twin of Attribute(RuleWord) in app/internal/segmentparts, held to it by a parity test: a speaker Correction or a
-- corrected text gives a single voice (ADR 0012), an ignored voice is no voice (ADR 0031), and only the voices of
-- the track of the segment are read (ADR 0052). One row per voice holding a part, in order of first part; a single
-- row with a NULL id when the segment has no voice. dominant marks the voice of the per-segment rule.
-- +goose StatementBegin
CREATE FUNCTION segment_part_speakers(p_clip bigint, p_stream int, p_start double precision, p_end double precision,
    p_job bigint, p_idx int, p_words jsonb) RETURNS TABLE(id bigint, dominant boolean, part_order int)
LANGUAGE plpgsql STABLE ROWS 2 AS $$
DECLARE
    ids bigint[];
    starts double precision[];
    ends double precision[];
    overriding bigint;
    text_corrected boolean := false;
    segment_speaker bigint;
    speakers bigint[] := '{}';
    previous bigint;
    neighbour bigint;
    chosen bigint;
    seen bigint[] := '{}';
    word jsonb;
    n int;
    i int;
BEGIN
    SELECT ls.id, sc.text IS NOT NULL INTO overriding, text_corrected
    FROM segment_corrections sc
    LEFT JOIN local_speakers ls ON ls.id = sc.speaker_override_id AND ls.kind <> 'archived' AND ls.clip_id = p_clip
        AND ls.audio_stream_index = p_stream
        AND NOT EXISTS (SELECT 1 FROM local_speaker_dispositions dsp WHERE dsp.local_speaker_id = ls.id AND dsp.disposition = 'ignored')
    WHERE sc.transcription_job_id = p_job AND sc.segment_index = p_idx AND sc.stale_at IS NULL;
    IF overriding IS NOT NULL THEN
        RETURN QUERY SELECT overriding, true, 0;
        RETURN;
    END IF;
    SELECT array_agg(t.speaker ORDER BY t.ordinal, t.speaker, t.turn_index), array_agg(t.start_seconds ORDER BY t.ordinal, t.speaker, t.turn_index),
        array_agg(t.end_seconds ORDER BY t.ordinal, t.speaker, t.turn_index) INTO ids, starts, ends
    FROM (SELECT ls.id speaker, ls.ordinal, st.turn_index, st.start_seconds, st.end_seconds
        FROM speaker_turns st JOIN local_speakers ls ON ls.id = st.local_speaker_id
        WHERE ls.clip_id = p_clip AND ls.audio_stream_index = p_stream AND ls.kind <> 'archived'
        AND NOT EXISTS (SELECT 1 FROM local_speaker_dispositions dsp WHERE dsp.local_speaker_id = ls.id AND dsp.disposition = 'ignored')) t;
    segment_speaker := dominant_turn_speaker(ids, starts, ends, p_start, p_end, NULL);
    IF NOT COALESCE(text_corrected, false) AND jsonb_typeof(p_words) = 'array' THEN
        FOR word IN SELECT w.value FROM jsonb_array_elements(p_words) WITH ORDINALITY w(value, position) ORDER BY w.position LOOP
            chosen := dominant_turn_speaker(ids, starts, ends, (word->>'start')::double precision, (word->>'end')::double precision, previous);
            speakers := array_append(speakers, chosen);
            previous := COALESCE(chosen, previous);
        END LOOP;
    END IF;
    n := COALESCE(array_length(speakers, 1), 0);
    IF previous IS NULL THEN
        RETURN QUERY SELECT segment_speaker, true, 0;
        RETURN;
    END IF;
    -- A word without turn takes its neighbour: the previous covered word, else the next one.
    FOR i IN 1..n LOOP
        IF speakers[i] IS NOT NULL THEN
            neighbour := speakers[i];
        ELSIF neighbour IS NOT NULL THEN
            speakers[i] := neighbour;
        END IF;
    END LOOP;
    FOR i IN REVERSE n..1 LOOP
        IF speakers[i] IS NOT NULL THEN
            neighbour := speakers[i];
        ELSE
            speakers[i] := neighbour;
        END IF;
    END LOOP;
    -- A single word between two words of one other voice is no change of voice.
    FOR i IN 2..n - 1 LOOP
        IF speakers[i - 1] = speakers[i + 1] AND speakers[i] <> speakers[i - 1] THEN
            speakers[i] := speakers[i - 1];
        END IF;
    END LOOP;
    FOR i IN 1..n LOOP
        IF NOT speakers[i] = ANY(seen) THEN
            RETURN QUERY SELECT speakers[i], COALESCE(speakers[i] = segment_speaker, false), COALESCE(array_length(seen, 1), 0);
            seen := array_append(seen, speakers[i]);
        END IF;
    END LOOP;
END $$;
-- +goose StatementEnd

-- +goose Down
DROP FUNCTION segment_part_speakers(bigint, int, double precision, double precision, bigint, int, jsonb);
DROP FUNCTION dominant_turn_speaker(bigint[], double precision[], double precision[], double precision, double precision, bigint);
DROP TABLE speaker_attribution_rules;
