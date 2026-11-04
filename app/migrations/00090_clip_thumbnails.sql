-- +goose Up
-- Miniature d'un clip indexé : un petit JPEG tiré par ffmpeg une seule fois (à l'indexation, ou à la
-- première demande), puis conservé. Elle disparaît avec le clip. Aucune miniature n'existe pour une
-- vidéo non indexée : lister ne lit pas le contenu des fichiers (ADR 0032).
CREATE TABLE clip_thumbnails (
    clip_id bigint PRIMARY KEY REFERENCES clips(id) ON DELETE CASCADE,
    jpeg bytea NOT NULL CHECK (length(jpeg) > 0),
    created_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE clip_thumbnails;
