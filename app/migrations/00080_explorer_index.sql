-- +goose Up
-- A file opened from the explorer (#195, ADR 0032) is indexed without being transcribed: its clip is
-- created, and nothing is analysed unless someone launches the analysis from the clip page.
ALTER TABLE source_files ADD COLUMN index_only boolean NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE source_files DROP COLUMN index_only;
