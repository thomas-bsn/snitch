-- +goose Up
-- The logo of the game a folder is named after, fetched once from the Steam store and kept here, so that browsing
-- the library never calls out. A name that matches no game is remembered too (found = false), and asked again
-- after a week. The key is the folder name reduced to letters and digits.
CREATE TABLE folder_logos (
    name_key text PRIMARY KEY CHECK (length(name_key) BETWEEN 3 AND 255),
    found boolean NOT NULL,
    content_type text NOT NULL DEFAULT '',
    image bytea,
    checked_at timestamptz NOT NULL DEFAULT now(),
    CHECK (found = (image IS NOT NULL))
);

-- +goose Down
DROP TABLE folder_logos;
