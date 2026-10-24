-- +goose Up
CREATE TABLE uploads (
 id text PRIMARY KEY CHECK (id ~ '^[0-9a-f]{32}$'),
 owner_id bigint NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 original_name text NOT NULL,
 total_bytes bigint NOT NULL CHECK (total_bytes > 0),
 offset_bytes bigint NOT NULL DEFAULT 0 CHECK (offset_bytes >= 0 AND offset_bytes <= total_bytes),
 visibility text NOT NULL CHECK (visibility IN ('private','shared')),
 status text NOT NULL CHECK (status IN ('uploading','processing','complete','failed','cancelled','expired')),
 clip_id bigint REFERENCES clips(id) ON DELETE RESTRICT,
 expires_at timestamptz NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now(),
 updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX uploads_owner_active ON uploads(owner_id,created_at DESC) WHERE status IN ('uploading','processing','failed');
CREATE INDEX uploads_expiry ON uploads(expires_at) WHERE status IN ('uploading','processing');

CREATE TABLE uploaded_media (
 upload_id text PRIMARY KEY REFERENCES uploads(id) ON DELETE RESTRICT,
 clip_id bigint NOT NULL REFERENCES clips(id) ON DELETE RESTRICT,
 owner_id bigint NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
 storage_key text NOT NULL UNIQUE CHECK (storage_key ~ '^[0-9a-f]{32}$'),
 original_name text NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX uploaded_media_clip ON uploaded_media(clip_id);
CREATE INDEX uploaded_media_owner ON uploaded_media(owner_id,created_at DESC);

-- +goose Down
DROP TABLE uploaded_media;
DROP TABLE uploads;
