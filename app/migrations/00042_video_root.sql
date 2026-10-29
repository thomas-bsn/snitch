-- +goose Up
ALTER TABLE clip_sources ADD COLUMN managed_root boolean NOT NULL DEFAULT false;
CREATE UNIQUE INDEX one_managed_video_root ON clip_sources(managed_root) WHERE managed_root;
ALTER TABLE source_files ADD COLUMN scan_requested boolean NOT NULL DEFAULT false;
-- +goose Down
ALTER TABLE source_files DROP COLUMN scan_requested;
DROP INDEX one_managed_video_root;
ALTER TABLE clip_sources DROP COLUMN managed_root;
