-- +goose Up
-- The folder-name matching now removes scene tags, edition suffixes and roman numerals, and searches Steam
-- with the reduced name. The misses remembered by the previous matching would hide those new matches for a
-- week, so they are dropped once; the positives stay valid.
DELETE FROM folder_logos WHERE NOT found;
-- An explicit app id (FOLDER_LOGOS_OVERRIDES) is kept with the image, so that changing the override is not
-- masked by the previous cache entry of the same folder name.
ALTER TABLE folder_logos ADD COLUMN app_id bigint NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE folder_logos DROP COLUMN app_id;
-- Nothing to restore: the deleted rows were only negative cache entries.
