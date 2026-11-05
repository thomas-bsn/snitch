-- +goose Up
-- When a Worker last asked for a job (issue #321). Presence (the pulse) says a Worker is Online; only this says
-- that its claiming loop works. An Online, idle Worker that stopped claiming is diagnosed instead of being shown
-- as busy. Existing Workers start from now, so a claiming loop already dead before this migration is caught too.
ALTER TABLE workers ADD COLUMN last_claim_at timestamptz NOT NULL DEFAULT now();

-- +goose Down
ALTER TABLE workers DROP COLUMN last_claim_at;
