-- +goose Up
-- « Scanner toute la racine » (#404, ADR 0038 amendée). The scan of the whole Racine vidéo becomes a run of
-- library_index_runs (mode scan): the request only records it, the walk of the disk happens in the background
-- and starts over after a restart (walking stays true until it ends). Every eligible video is tagged with the
-- run, the managed scan then indexes it and the analysis follows the processing settings. The counters are the
-- walk's own findings, rewritten as it goes: what was already up to date and what was left out, by reason.
ALTER TABLE library_index_runs DROP CONSTRAINT library_index_runs_mode_check;
ALTER TABLE library_index_runs ADD CONSTRAINT library_index_runs_mode_check CHECK (mode IN ('cards','full','scan'));
ALTER TABLE library_index_runs
    ADD COLUMN walking boolean NOT NULL DEFAULT false,
    -- Eligible videos met by the walk so far.
    ADD COLUMN seen integer NOT NULL DEFAULT 0 CHECK (seen >= 0),
    -- Indexed videos whose clip was already analysed (or is being analysed): nothing to queue.
    ADD COLUMN up_to_date integer NOT NULL DEFAULT 0 CHECK (up_to_date >= 0),
    ADD COLUMN excluded_private_folders integer NOT NULL DEFAULT 0 CHECK (excluded_private_folders >= 0),
    ADD COLUMN excluded_hidden integer NOT NULL DEFAULT 0 CHECK (excluded_hidden >= 0),
    ADD COLUMN excluded_system integer NOT NULL DEFAULT 0 CHECK (excluded_system >= 0),
    ADD COLUMN excluded_private_clips integer NOT NULL DEFAULT 0 CHECK (excluded_private_clips >= 0),
    ADD COLUMN unreadable_folders integer NOT NULL DEFAULT 0 CHECK (unreadable_folders >= 0),
    -- The first unreadable folders, relative to the Racine, so the report can name them.
    ADD COLUMN unreadable_paths text[] NOT NULL DEFAULT '{}',
    ADD CONSTRAINT library_index_runs_walk_scan CHECK (NOT walking OR (mode = 'scan' AND status = 'running'));

-- +goose Down
DELETE FROM library_index_runs WHERE mode = 'scan';
ALTER TABLE library_index_runs
    DROP CONSTRAINT library_index_runs_walk_scan,
    DROP COLUMN unreadable_paths,
    DROP COLUMN unreadable_folders,
    DROP COLUMN excluded_private_clips,
    DROP COLUMN excluded_system,
    DROP COLUMN excluded_hidden,
    DROP COLUMN excluded_private_folders,
    DROP COLUMN up_to_date,
    DROP COLUMN seen,
    DROP COLUMN walking;
ALTER TABLE library_index_runs DROP CONSTRAINT library_index_runs_mode_check;
ALTER TABLE library_index_runs ADD CONSTRAINT library_index_runs_mode_check CHECK (mode IN ('cards','full'));
