-- +goose Up
-- Keep logo identity separate from selection: switching between retained logos must not create history.
ALTER TABLE folder_logo_choices ADD COLUMN active_version integer;
ALTER TABLE folder_logo_choices ADD CONSTRAINT folder_logo_active_version_fk
    FOREIGN KEY (source_id, relative_path, active_version)
    REFERENCES folder_logo_versions(source_id, relative_path, version)
    DEFERRABLE INITIALLY DEFERRED;

-- Existing instances used the newest retained version as the active logo.
UPDATE folder_logo_choices c SET active_version = (
    SELECT MAX(v.version) FROM folder_logo_versions v
    WHERE v.source_id = c.source_id AND v.relative_path = c.relative_path
);

-- +goose Down
ALTER TABLE folder_logo_choices DROP CONSTRAINT folder_logo_active_version_fk;
ALTER TABLE folder_logo_choices DROP COLUMN active_version;
