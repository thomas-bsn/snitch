-- +goose Up
CREATE TABLE clip_deletion_files (
 clip_id bigint NOT NULL REFERENCES clips(id) ON DELETE CASCADE,
 root_path text NOT NULL,
 relative_path text NOT NULL,
 staged_path text NOT NULL,
 phase text NOT NULL CHECK (phase IN ('planned', 'moving', 'removing', 'removed')),
 PRIMARY KEY (clip_id, root_path, relative_path)
);
-- +goose Down
DROP TABLE clip_deletion_files;
