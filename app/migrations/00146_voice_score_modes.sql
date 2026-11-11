-- +goose Up
-- Score mode of voice matching (ADR 0064, #391). The Reference of a Person is cumulative by default: the mean of
-- its contributive validated samples weighted by their net speech. The phase 5 score, the best cosine with one
-- comparable sample, stays available so the choice can be measured on the installation and reversed.
--
-- Append-only history: the active mode is the highest version. Version 1 is Snitch's default (no author).
-- Authors are snapshots, without foreign key: accounts may disappear.
CREATE TABLE voice_score_modes (
    version integer PRIMARY KEY CHECK (version >= 1),
    mode text NOT NULL CHECK (mode IN ('cumulative', 'best_sample')),
    author_id bigint,
    author_username text,
    created_at timestamptz NOT NULL DEFAULT now()
);
INSERT INTO voice_score_modes(version, mode) VALUES (1, 'cumulative');

-- +goose Down
DROP TABLE voice_score_modes;
