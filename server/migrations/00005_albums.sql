-- +goose Up
-- Album domain: albums of all three types, materialized membership, view
-- ACLs, per-user timeline preferences, favorites, and tags.

CREATE TABLE albums (
    id              uuid PRIMARY KEY,
    owner_id        uuid NOT NULL REFERENCES users (id),
    title           text NOT NULL,
    description     text NOT NULL DEFAULT '',
    type            album_type NOT NULL,
    locked          boolean NOT NULL DEFAULT false,
    cover_asset_id  uuid,
    root_path       text,
    last_scanned_at timestamptz,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now()
);

-- The sync album is the user's core library stream, so each user may own at
-- most one; directory and virtual albums are unlimited.
CREATE UNIQUE INDEX albums_sync_owner_uq
    ON albums (owner_id)
    WHERE type = 'sync';

CREATE TABLE album_items (
    album_id uuid NOT NULL REFERENCES albums (id) ON DELETE CASCADE,
    asset_id uuid NOT NULL REFERENCES assets (id) ON DELETE CASCADE,
    added_by uuid NOT NULL REFERENCES users (id),
    added_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (album_id, asset_id)
);

CREATE INDEX album_items_asset_idx ON album_items (asset_id);

CREATE TABLE album_shares (
    album_id   uuid NOT NULL REFERENCES albums (id) ON DELETE CASCADE,
    user_id    uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    granted_by uuid NOT NULL REFERENCES users (id),
    granted_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (album_id, user_id)
);

CREATE INDEX album_shares_user_idx ON album_shares (user_id);

CREATE TABLE user_album_prefs (
    user_id     uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    album_id    uuid NOT NULL REFERENCES albums (id) ON DELETE CASCADE,
    in_timeline boolean NOT NULL DEFAULT true,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, album_id)
);

CREATE TABLE user_favorites (
    user_id  uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    asset_id uuid NOT NULL REFERENCES assets (id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, asset_id)
);

CREATE TABLE tags (
    id         uuid PRIMARY KEY,
    owner_id   uuid NOT NULL REFERENCES users (id),
    name       text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (owner_id, name)
);

CREATE TABLE asset_tags (
    asset_id   uuid NOT NULL REFERENCES assets (id) ON DELETE CASCADE,
    tag_id     uuid NOT NULL REFERENCES tags (id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (asset_id, tag_id)
);

-- +goose Down
DROP TABLE IF EXISTS asset_tags;
DROP TABLE IF EXISTS tags;
DROP TABLE IF EXISTS user_favorites;
DROP TABLE IF EXISTS user_album_prefs;
DROP TABLE IF EXISTS album_shares;
DROP TABLE IF EXISTS album_items;
DROP TABLE IF EXISTS albums;
