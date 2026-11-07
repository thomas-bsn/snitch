-- +goose Up
-- A custom logo for a folder of the video root: chosen by an administrator, kept locally, and identified
-- by the source and the folder path rather than by its name (two folders may share a name). `use_custom`
-- tells whether the custom image is shown; setting it to false restores the automatic Steam cover without
-- destroying the history.
CREATE TABLE folder_logo_choices (
    source_id bigint NOT NULL REFERENCES clip_sources(id) ON DELETE CASCADE,
    relative_path text NOT NULL CHECK (relative_path <> '' AND length(relative_path) <= 1024),
    use_custom boolean NOT NULL DEFAULT true,
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (source_id, relative_path)
);

-- The current custom logo and its three previous images. Restoring an older image creates a new version
-- from it, as Discord does; a stored version is never modified in place.
CREATE TABLE folder_logo_versions (
    id bigserial PRIMARY KEY,
    source_id bigint NOT NULL,
    relative_path text NOT NULL,
    version integer NOT NULL CHECK (version > 0),
    image bytea NOT NULL,
    content_type text NOT NULL CHECK (content_type IN ('image/png','image/jpeg')),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (source_id, relative_path, version),
    FOREIGN KEY (source_id, relative_path) REFERENCES folder_logo_choices(source_id, relative_path) ON DELETE CASCADE
);
CREATE INDEX folder_logo_versions_recent ON folder_logo_versions(source_id, relative_path, version DESC);

-- +goose Down
DROP TABLE folder_logo_versions, folder_logo_choices;
