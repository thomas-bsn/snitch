-- +goose Up
-- Version of the effective text of a segment, distinct from the correction version
-- (which every write increments, speaker included). Only text writes and restorations
-- increment it. Indexing and reindexing compare this one: a speaker correction made
-- while a segment is being indexed does not change its text, so it must not make the
-- indexing skip the segment (the speaker does not request any reindexing).
ALTER TABLE segment_corrections ADD COLUMN text_version integer NOT NULL DEFAULT 0 CHECK (text_version >= 0);
-- Existing rows: every version so far is conservatively treated as a text version.
UPDATE segment_corrections SET text_version = version;

-- +goose Down
ALTER TABLE segment_corrections DROP COLUMN text_version;
