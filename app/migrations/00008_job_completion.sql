-- +goose Up
ALTER TABLE jobs ADD COLUMN completed_by_worker_id text REFERENCES workers(id) ON DELETE SET NULL;
ALTER TABLE jobs ADD COLUMN lease_token_hash bytea CHECK (lease_token_hash IS NULL OR octet_length(lease_token_hash) = 32);
ALTER TABLE jobs ADD COLUMN completed_lease_token_hash bytea CHECK (completed_lease_token_hash IS NULL OR octet_length(completed_lease_token_hash) = 32);
ALTER TABLE jobs DROP CONSTRAINT jobs_check;
UPDATE jobs SET status='pending',lease_owner=NULL,lease_expires_at=NULL WHERE status='leased';
UPDATE jobs SET lease_owner=NULL,lease_expires_at=NULL,lease_token_hash=NULL WHERE status<>'leased';
ALTER TABLE jobs ADD CONSTRAINT jobs_lease_state_check CHECK (
  (status='leased' AND lease_owner IS NOT NULL AND lease_expires_at IS NOT NULL AND lease_token_hash IS NOT NULL)
  OR (status<>'leased' AND lease_owner IS NULL AND lease_expires_at IS NULL AND lease_token_hash IS NULL)
);

-- +goose Down
ALTER TABLE jobs DROP CONSTRAINT jobs_lease_state_check;
ALTER TABLE jobs ADD CONSTRAINT jobs_check CHECK ((status = 'leased') = (lease_owner IS NOT NULL AND lease_expires_at IS NOT NULL));
ALTER TABLE jobs DROP COLUMN completed_lease_token_hash;
ALTER TABLE jobs DROP COLUMN lease_token_hash;
ALTER TABLE jobs DROP COLUMN completed_by_worker_id;
