-- +goose Up
CREATE TABLE upload_destinations (
 upload_id text PRIMARY KEY REFERENCES uploads(id) ON DELETE CASCADE,
 source_id bigint NOT NULL REFERENCES clip_sources(id) ON DELETE RESTRICT,
 relative_path text NOT NULL,
 expected_version text NOT NULL,
 published boolean NOT NULL DEFAULT false
);
-- +goose Down
DROP TABLE upload_destinations;
