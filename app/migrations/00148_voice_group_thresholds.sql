-- +goose Up
-- Matching thresholds of a fingerprint group (ADR 0064, #393). The scale of a cosine score depends on the
-- embedding model, so the suggestion threshold, the automatic threshold and the ambiguity margin belong to one
-- (model, model_version, dimensions) group. A group without applied thresholds only produces suggestions: after
-- a change of fingerprint model, automatic attribution stays suspended until an administrator applies
-- thresholds to the new group, from a measure or by hand.
--
-- Append-only history: the thresholds of a group are its highest version. source tells where they come from:
-- 'migration' (the installation's thresholds taken over below), 'administrator' (typed by hand) or 'proposal'
-- (the proposal of the measured accuracy, applied by hand). score_mode is the score mode active when they were
-- applied. Authors are snapshots without foreign key (NULL: Snitch).
--
-- The values of voice_matching_settings stay as provisional values: the suggestion threshold and the margin of
-- a group without applied thresholds.
CREATE TABLE voice_group_thresholds (
    id bigserial PRIMARY KEY,
    model text NOT NULL CHECK (model <> ''),
    model_version text NOT NULL CHECK (model_version <> ''),
    dimensions integer NOT NULL CHECK (dimensions BETWEEN 1 AND 2000),
    version integer NOT NULL CHECK (version >= 1),
    suggestion_threshold double precision NOT NULL,
    automatic_threshold double precision NOT NULL,
    ambiguity_margin double precision NOT NULL,
    source text NOT NULL CHECK (source IN ('migration', 'administrator', 'proposal')),
    score_mode text NOT NULL CHECK (score_mode IN ('cumulative', 'best_sample')),
    author_id bigint,
    author_username text,
    created_at timestamptz NOT NULL DEFAULT now(),
    CHECK (suggestion_threshold >= 0 AND suggestion_threshold <= automatic_threshold AND automatic_threshold <= 1),
    CHECK (ambiguity_margin >= 0 AND ambiguity_margin <= 1),
    UNIQUE (model, model_version, dimensions, version)
);

-- The thresholds of the installation go to the active group, and to no other: the group of the latest usable
-- production speaker fingerprints of the active fingerprint model or, when that model has none, the group with
-- the most production speaker fingerprints (models outside the catalogue). A blank installation has no group and
-- gets nothing; any other group starts without thresholds. Kept identical to activeFingerprintGroup
-- (voice_active_group.go).
INSERT INTO voice_group_thresholds(model, model_version, dimensions, version, suggestion_threshold, automatic_threshold, ambiguity_margin, source, score_mode)
WITH candidates AS (
    (SELECT 0 AS priority, f.model, f.model_version, f.dimensions FROM voice_fingerprints f
     WHERE f.usable AND NOT f.benchmark AND f.scope = 'speaker'
       AND f.model = (SELECT m.repo FROM voice_active_model a JOIN voice_model_catalog m ON m.id = a.model_id WHERE a.kind = 'fingerprint')
     GROUP BY f.model, f.model_version, f.dimensions ORDER BY max(f.id) DESC LIMIT 1)
    UNION ALL
    (SELECT 1, f.model, f.model_version, f.dimensions FROM voice_fingerprints f
     WHERE NOT f.benchmark AND f.scope = 'speaker'
     GROUP BY f.model, f.model_version, f.dimensions ORDER BY count(*) DESC, max(f.id) DESC LIMIT 1)
)
SELECT g.model, g.model_version, g.dimensions, 1, s.suggestion_threshold, s.automatic_threshold, s.ambiguity_margin, 'migration',
       (SELECT mode FROM voice_score_modes ORDER BY version DESC LIMIT 1)
FROM (SELECT * FROM candidates ORDER BY priority LIMIT 1) g
CROSS JOIN voice_matching_settings s
WHERE s.id AND NOT EXISTS (SELECT 1 FROM voice_group_thresholds);

-- +goose Down
DROP TABLE voice_group_thresholds;
