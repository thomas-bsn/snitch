-- +goose Up
-- An optional label chosen by an administrator, kept apart from the physical file name (original_name is never
-- touched). detected_metadata keeps the automatically detected game and capture time the first time they are
-- edited by hand, so they can be restored; a rescan only fills what was not edited.
ALTER TABLE clips ADD COLUMN display_title text CHECK (display_title IS NULL OR char_length(display_title) BETWEEN 1 AND 200);
ALTER TABLE clips ADD COLUMN detected_metadata jsonb;
-- +goose Down
ALTER TABLE clips DROP COLUMN detected_metadata, DROP COLUMN display_title;
