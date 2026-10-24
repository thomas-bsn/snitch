-- +goose Up
CREATE TABLE clip_sources (
 id bigserial PRIMARY KEY,
 name text NOT NULL,
 root_path text NOT NULL,
 owner_id bigint NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
 visibility text NOT NULL DEFAULT 'shared' CHECK (visibility IN ('private','shared')),
 enabled boolean NOT NULL DEFAULT true,
 health text NOT NULL DEFAULT 'ready' CHECK (health IN ('ready','unavailable','warning')),
 last_scan_at timestamptz,
 scan_generation bigint NOT NULL DEFAULT 0,
 created_at timestamptz NOT NULL DEFAULT now(),
 UNIQUE(root_path),
 UNIQUE(name)
);

CREATE TABLE clips (
 id bigserial PRIMARY KEY,
 owner_id bigint NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
 visibility text NOT NULL DEFAULT 'shared' CHECK (visibility IN ('private','shared')),
 original_name text NOT NULL,
 size_bytes bigint NOT NULL CHECK (size_bytes >= 0),
 modified_at timestamptz NOT NULL,
 discovered_at timestamptz NOT NULL DEFAULT now(),
 recorded_at timestamptz NOT NULL,
 recorded_at_estimated boolean NOT NULL DEFAULT true,
 duration_seconds double precision,
 format_name text NOT NULL DEFAULT '',
 video_codec text NOT NULL DEFAULT '',
 audio_codec text NOT NULL DEFAULT '',
 tracks jsonb NOT NULL DEFAULT '[]'::jsonb
);

CREATE TABLE source_files (
 source_id bigint NOT NULL REFERENCES clip_sources(id) ON DELETE RESTRICT,
 relative_path text NOT NULL,
 clip_id bigint REFERENCES clips(id) ON DELETE RESTRICT,
 size_bytes bigint NOT NULL CHECK (size_bytes >= 0),
 modified_at timestamptz NOT NULL,
 stable_since timestamptz NOT NULL DEFAULT now(),
 last_seen_generation bigint NOT NULL,
 status text NOT NULL CHECK (status IN ('waiting_stability','ready','unreadable','missing')),
 error_code text,
 PRIMARY KEY(source_id,relative_path)
);

CREATE INDEX source_files_clip ON source_files(clip_id) WHERE clip_id IS NOT NULL;
CREATE INDEX source_files_state ON source_files(source_id,status);
CREATE INDEX clips_recorded_at ON clips(recorded_at DESC);

-- +goose Down
DROP TABLE source_files, clips, clip_sources;
