-- +goose Up
-- User domain: accounts, sessions, pairing tokens, API keys, and
-- instance-wide settings.

CREATE TABLE users (
    id            uuid PRIMARY KEY,
    name          text NOT NULL,
    email         text NOT NULL UNIQUE,
    password_hash text NOT NULL,
    role          user_role NOT NULL DEFAULT 'user',
    locale        text NOT NULL DEFAULT 'en',
    theme         text NOT NULL DEFAULT 'auto',
    is_disabled   boolean NOT NULL DEFAULT false,
    totp_secret   text,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE sessions (
    id           uuid PRIMARY KEY,
    user_id      uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token_hash   text NOT NULL UNIQUE,
    kind         session_kind NOT NULL,
    device_name  text NOT NULL DEFAULT '',
    device_id    text,
    last_seen_at timestamptz,
    expires_at   timestamptz NOT NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE pairing_tokens (
    id         uuid PRIMARY KEY,
    token_hash text NOT NULL UNIQUE,
    created_by uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    expires_at timestamptz NOT NULL,
    used_at    timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE api_keys (
    id           uuid PRIMARY KEY,
    user_id      uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    name         text NOT NULL,
    key_hash     text NOT NULL UNIQUE,
    last_used_at timestamptz,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE instance_settings (
    key        text PRIMARY KEY,
    value      jsonb NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS pairing_tokens;
DROP TABLE IF EXISTS api_keys;
DROP TABLE IF EXISTS instance_settings;
DROP TABLE IF EXISTS users;
