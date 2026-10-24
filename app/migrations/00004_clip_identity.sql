-- +goose Up
ALTER TABLE clips ADD COLUMN content_sha256 bytea;
CREATE UNIQUE INDEX clips_content_sha256_key ON clips(content_sha256);

ALTER TABLE clip_sources ADD COLUMN health_message text NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE clip_sources DROP COLUMN health_message;
DROP INDEX clips_content_sha256_key;
ALTER TABLE clips DROP COLUMN content_sha256;
