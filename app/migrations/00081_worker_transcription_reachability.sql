-- +goose Up
-- Dernier résultat de joignabilité du serveur de transcription, transmis par chaque Worker dans son pouls (#212).
-- reach_checked_at NULL : aucun test reçu (inconnu). reach_failure : détail du dernier échec pour la cible courante.
ALTER TABLE worker_transcription_config
    ADD COLUMN reachable boolean NOT NULL DEFAULT false,
    ADD COLUMN reach_checked_at timestamptz,
    ADD COLUMN reach_latency_ms integer NOT NULL DEFAULT 0,
    ADD COLUMN reach_failure jsonb;

-- +goose Down
ALTER TABLE worker_transcription_config
    DROP COLUMN reach_failure,
    DROP COLUMN reach_latency_ms,
    DROP COLUMN reach_checked_at,
    DROP COLUMN reachable;
