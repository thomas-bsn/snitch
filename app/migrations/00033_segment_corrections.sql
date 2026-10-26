-- +goose Up
-- Corrections of transcript segments (phase 6). The worker transcription stored in
-- jobs.result is never modified: a correction is a separate layer keyed by the
-- segment identity (transcription job, index in result.segments).
--
-- This single migration lays every column the later phase 6 steps need, so they do
-- not have to migrate: history/restore (events), optimistic concurrency (version),
-- targeted reindexing (reindex_*), speaker override (speaker_*), retranscription
-- (stale_at). See docs/segment-corrections.md.

-- Current state of a segment correction. A segment without row has version 0 and no
-- correction. Rows disappear only with their job or clip (cascade).
CREATE TABLE segment_corrections (
    transcription_job_id bigint NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    segment_index integer NOT NULL CHECK (segment_index >= 0),
    clip_id bigint NOT NULL REFERENCES clips(id) ON DELETE CASCADE,
    -- Corrected text; NULL means the original text applies (never corrected, or restored).
    text text CHECK (text IS NULL OR (btrim(text) <> '' AND char_length(text) <= 4000)),
    text_author_id bigint,
    text_author_username text NOT NULL DEFAULT '',
    text_corrected_at timestamptz,
    -- Speaker override (reserved for the speaker step); NULL means the deduced speaker.
    speaker_override_id bigint REFERENCES local_speakers(id) ON DELETE SET NULL,
    speaker_author_id bigint,
    speaker_author_username text NOT NULL DEFAULT '',
    speaker_corrected_at timestamptz,
    -- Incremented by every write to the row (text, speaker or restoration): the base of
    -- the optimistic concurrency check. The first write creates the row at version 1.
    version integer NOT NULL DEFAULT 1 CHECK (version >= 1),
    -- Set when a retranscription replaced the transcript this correction belongs to;
    -- an obsolete correction is never applied again.
    stale_at timestamptz,
    -- Semantic reindexing marker: set by a text correction, cleared by the reindexing step.
    reindex_pending boolean NOT NULL DEFAULT false,
    reindex_requested_at timestamptz,
    reindex_attempts integer NOT NULL DEFAULT 0 CHECK (reindex_attempts >= 0),
    reindex_error text NOT NULL DEFAULT '',
    reindex_lease_expires_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (transcription_job_id, segment_index)
);
CREATE INDEX segment_corrections_clip ON segment_corrections(clip_id);
CREATE INDEX segment_corrections_reindex ON segment_corrections(reindex_requested_at) WHERE reindex_pending;
CREATE INDEX segment_corrections_speaker ON segment_corrections(speaker_override_id) WHERE speaker_override_id IS NOT NULL;

-- Append-only history. Texts and speakers are stored in full (effective value before
-- and after), author by id and name without foreign key so the trail survives account
-- deletions. Rows disappear only through the cascade from their job or clip; direct
-- UPDATE/DELETE is refused by the same trigger function as the attribution history.
CREATE TABLE segment_correction_events (
    id bigserial PRIMARY KEY,
    transcription_job_id bigint NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    segment_index integer NOT NULL CHECK (segment_index >= 0),
    clip_id bigint NOT NULL REFERENCES clips(id) ON DELETE CASCADE,
    kind text NOT NULL CHECK (kind IN ('text','speaker','restore')),
    -- Effective text before/after (original text when no correction applied).
    old_text text,
    new_text text,
    old_corrected boolean NOT NULL DEFAULT false,
    new_corrected boolean NOT NULL DEFAULT false,
    -- Speaker step: local speaker id and label at the time; NULL = deduced speaker.
    old_speaker_id bigint,
    old_speaker_label text NOT NULL DEFAULT '',
    new_speaker_id bigint,
    new_speaker_label text NOT NULL DEFAULT '',
    -- Restoration: source event (no foreign key: immutable rows reference by value) or the original text.
    restored_from_event_id bigint,
    restored_to_original boolean NOT NULL DEFAULT false,
    reason text NOT NULL DEFAULT '' CHECK (char_length(reason) <= 500),
    version_before integer NOT NULL CHECK (version_before >= 0),
    version_after integer NOT NULL CHECK (version_after >= 1),
    author_id bigint,
    author_username text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX segment_correction_events_segment ON segment_correction_events(transcription_job_id, segment_index, id);
CREATE INDEX segment_correction_events_clip ON segment_correction_events(clip_id, id);
CREATE TRIGGER segment_correction_events_immutable BEFORE UPDATE OR DELETE ON segment_correction_events
    FOR EACH ROW EXECUTE FUNCTION attribution_events_immutable();

-- +goose Down
DROP TABLE segment_correction_events;
DROP TABLE segment_corrections;
