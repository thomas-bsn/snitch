-- +goose Up
-- A private folder (#93): a top-level folder of a source that only the Super-administrator may browse.
-- Its name stays listed (with a lock) for everyone else, its clips and paths do not. This is distinct
-- from a clip's own visibility: both rules apply, and a clip's owner always keeps access to it.
CREATE TABLE private_folders (
 source_id bigint NOT NULL REFERENCES clip_sources(id) ON DELETE CASCADE,
 folder text NOT NULL CHECK (folder <> '' AND folder NOT IN ('.','..') AND position('/' in folder)=0),
 set_by bigint REFERENCES users(id) ON DELETE SET NULL,
 created_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY(source_id,folder)
);

-- +goose StatementBegin
-- Whether a path relative to a source lies inside one of its private top-level folders.
-- A file at the root of a source is in no folder, so it is never private.
CREATE FUNCTION path_in_private_folder(p_source bigint, p_path text) RETURNS boolean LANGUAGE sql STABLE AS $$
 SELECT position('/' in p_path)>0 AND EXISTS(SELECT 1 FROM private_folders pf WHERE pf.source_id=p_source AND pf.folder=split_part(p_path,'/',1))
$$;
-- +goose StatementEnd

-- +goose StatementBegin
-- Whether a clip may be opened by someone who cannot browse private folders. A clip referenced by no
-- private folder is open. One referenced by a private folder stays open only through at least one
-- other live provenance: a non-missing file outside any private folder, or an upload not destined to
-- one. A copy that went missing does not count, so moving a file into a private folder hides it.
CREATE FUNCTION clip_folder_open(p_clip bigint) RETURNS boolean LANGUAGE sql STABLE AS $$
 SELECT NOT (
   EXISTS(SELECT 1 FROM source_files sf WHERE sf.clip_id=p_clip AND path_in_private_folder(sf.source_id,sf.relative_path))
   OR EXISTS(SELECT 1 FROM uploaded_media um JOIN upload_destinations d ON d.upload_id=um.upload_id WHERE um.clip_id=p_clip AND path_in_private_folder(d.source_id,d.relative_path))
 )
 OR EXISTS(SELECT 1 FROM source_files sf WHERE sf.clip_id=p_clip AND sf.status<>'missing' AND NOT path_in_private_folder(sf.source_id,sf.relative_path))
 OR EXISTS(SELECT 1 FROM uploaded_media um WHERE um.clip_id=p_clip AND NOT EXISTS(SELECT 1 FROM upload_destinations d WHERE d.upload_id=um.upload_id AND path_in_private_folder(d.source_id,d.relative_path)))
$$;
-- +goose StatementEnd

-- +goose StatementBegin
-- The single rule deciding whether a user may read a clip: its owner and the Super-administrator
-- always may; anyone else needs the clip's own visibility (shared, or an administrator) AND an open
-- folder. Every listing, search, count and file endpoint goes through it.
CREATE FUNCTION clip_readable(p_clip bigint, p_owner bigint, p_visibility text, p_user bigint, p_admin boolean) RETURNS boolean LANGUAGE sql STABLE AS $$
 SELECT p_owner=p_user
  OR EXISTS(SELECT 1 FROM users WHERE id=p_user AND role='super_admin')
  OR ((p_visibility='shared' OR p_admin) AND clip_folder_open(p_clip))
$$;
-- +goose StatementEnd

-- +goose Down
DROP FUNCTION clip_readable(bigint,bigint,text,bigint,boolean);
DROP FUNCTION clip_folder_open(bigint);
DROP FUNCTION path_in_private_folder(bigint,text);
DROP TABLE private_folders;
