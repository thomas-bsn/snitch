-- +goose Up
-- Two images can now coexist per account: the personal one ("custom", every existing row) and the one
-- downloaded from Discord ("discord"). The account's choice lives in avatar_preferences.
ALTER TABLE account_avatars ADD COLUMN source text NOT NULL DEFAULT 'custom' CHECK (source IN ('custom','discord'));
ALTER TABLE account_avatars ADD COLUMN discord_hash text CHECK (discord_hash IS NULL OR source='discord');
ALTER TABLE account_avatars DROP CONSTRAINT account_avatars_pkey;
ALTER TABLE account_avatars ADD PRIMARY KEY (user_id, source);

CREATE TABLE avatar_preferences (
 user_id bigint PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
 source text NOT NULL DEFAULT 'none' CHECK (source IN ('discord','custom','none')),
 -- Bumped at every change of what the account displays; drives the HTTP validator of the image.
 version bigint NOT NULL DEFAULT 1,
 discord_checked_at timestamptz,
 discord_next_attempt_at timestamptz,
 discord_failures integer NOT NULL DEFAULT 0,
 discord_last_error text
);
-- Existing accounts that already had a photo keep it, as their explicit personal choice.
INSERT INTO avatar_preferences(user_id,source) SELECT user_id,'custom' FROM account_avatars;
-- Accounts already linked to Discord and without a personal photo adopt the Discord photo, as a new link would.
INSERT INTO avatar_preferences(user_id,source) SELECT id,'discord' FROM users WHERE discord_id IS NOT NULL AND deleted_at IS NULL AND id NOT IN (SELECT user_id FROM avatar_preferences);

-- One row: the state of the Discord bot integration (the token itself is a sealed setting).
CREATE TABLE discord_bot_state (
 id smallint PRIMARY KEY CHECK (id=1),
 bot_name text,
 status text NOT NULL DEFAULT 'connected' CHECK (status IN ('connected','error')),
 last_error text,
 last_sync_started_at timestamptz,
 last_sync_at timestamptz,
 last_refreshed integer NOT NULL DEFAULT 0,
 last_failed integer NOT NULL DEFAULT 0,
 -- Lease taken by the instance that runs a synchronization, so that several instances never overlap.
 locked_until timestamptz
);
INSERT INTO discord_bot_state(id) VALUES (1);
-- +goose Down
DROP TABLE discord_bot_state;
DROP TABLE avatar_preferences;
DELETE FROM account_avatars WHERE source='discord';
ALTER TABLE account_avatars DROP CONSTRAINT account_avatars_pkey;
ALTER TABLE account_avatars DROP COLUMN discord_hash;
ALTER TABLE account_avatars DROP COLUMN source;
ALTER TABLE account_avatars ADD PRIMARY KEY (user_id);
