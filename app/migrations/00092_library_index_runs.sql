-- +goose Up
-- Indexation de toute la Racine vidéo (#275, ADR 0038). An Administrator asks, once and
-- explicitly, for every video of the Racine to get its clip card (and optionally its full
-- analysis). The run only records the request and its mode; the files it targets are the
-- source_files rows tagged with its id, processed by the existing managed scan. Nothing here
-- runs without that explicit action (ADR 0017).
CREATE TABLE library_index_runs (
    id bigserial PRIMARY KEY,
    source_id bigint NOT NULL REFERENCES clip_sources(id) ON DELETE CASCADE,
    -- cards: clip cards only. full: clip cards, then the full analysis of each new clip.
    mode text NOT NULL CHECK (mode IN ('cards','full')),
    status text NOT NULL DEFAULT 'running' CHECK (status IN ('running','paused','cancelled','completed')),
    created_by bigint REFERENCES users(id) ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    finished_at timestamptz,
    CHECK ((status IN ('cancelled','completed')) = (finished_at IS NOT NULL))
);
-- One run in progress per Racine: a second request cannot double the work.
CREATE UNIQUE INDEX library_index_runs_active ON library_index_runs(source_id) WHERE status IN ('running','paused');

ALTER TABLE source_files ADD COLUMN index_run_id bigint REFERENCES library_index_runs(id) ON DELETE SET NULL;
CREATE INDEX source_files_index_run ON source_files(index_run_id) WHERE index_run_id IS NOT NULL;

-- +goose Down
DROP INDEX source_files_index_run;
ALTER TABLE source_files DROP COLUMN index_run_id;
DROP TABLE library_index_runs;
