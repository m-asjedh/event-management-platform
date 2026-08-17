-- Better Auth owns the logic, these migrations own the tables. One thing creates
-- schema, so `make up` is deterministic on a clean machine.
--
-- Two renames, both mapped back in the Better Auth config: its default table is `user`
-- (a reserved SQL word) and its default columns are camelCase (would need quoting in
-- every query).

-- +goose Up

CREATE SCHEMA IF NOT EXISTS auth;

-- id is text, not uuid: Better Auth generates the value itself.
CREATE TABLE auth.users (
    id             text        PRIMARY KEY,
    name           text        NOT NULL,
    email          text        NOT NULL UNIQUE,
    email_verified boolean     NOT NULL DEFAULT false,
    image          text,
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE auth.sessions (
    id         text        PRIMARY KEY,
    user_id    text        NOT NULL REFERENCES auth.users (id) ON DELETE CASCADE,
    token      text        NOT NULL UNIQUE,
    expires_at timestamptz NOT NULL,
    ip_address text,
    user_agent text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX sessions_user_id_idx ON auth.sessions (user_id);

-- The password hash lives here under the "credential" provider, not on users.
CREATE TABLE auth.accounts (
    id                       text        PRIMARY KEY,
    user_id                  text        NOT NULL REFERENCES auth.users (id) ON DELETE CASCADE,
    account_id               text        NOT NULL,
    provider_id              text        NOT NULL,
    access_token             text,
    refresh_token            text,
    id_token                 text,
    access_token_expires_at  timestamptz,
    refresh_token_expires_at timestamptz,
    scope                    text,
    password                 text,
    created_at               timestamptz NOT NULL DEFAULT now(),
    updated_at               timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX accounts_user_id_idx ON auth.accounts (user_id);

CREATE TABLE auth.verifications (
    id         text        PRIMARY KEY,
    identifier text        NOT NULL,
    value      text        NOT NULL,
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX verifications_identifier_idx ON auth.verifications (identifier);

-- +goose Down

DROP TABLE auth.verifications;
DROP TABLE auth.accounts;
DROP TABLE auth.sessions;
DROP TABLE auth.users;
DROP SCHEMA auth;
