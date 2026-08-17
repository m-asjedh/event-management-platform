-- Events, the rooms and sessions inside them, membership with a role, and invitations.

-- +goose Up

-- Lets the room exclusion constraint below mix a uuid equality test with a range
-- overlap test in one GiST index.
CREATE EXTENSION IF NOT EXISTS btree_gist;

-- A CHECK cannot query pg_timezone_names, so zone validity is a foreign key instead.
CREATE TABLE time_zones (
    name text PRIMARY KEY
);

INSERT INTO time_zones (name) SELECT name FROM pg_timezone_names;

-- +goose StatementBegin
CREATE FUNCTION set_updated_at() RETURNS trigger AS $$
BEGIN
    NEW.updated_at := now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

-- Instants are timestamptz, so they are absolute. time_zone is stored separately
-- because it belongs to the event: it turns an instant back into local wall-clock time,
-- and it is what "next Tuesday at 9am" resolves against.
CREATE TABLE events (
    id          uuid        PRIMARY KEY DEFAULT uuidv7(),
    name        text        NOT NULL,
    description text        NOT NULL DEFAULT '',
    time_zone   text        NOT NULL REFERENCES time_zones (name),
    starts_at   timestamptz NOT NULL,
    ends_at     timestamptz NOT NULL,
    version     integer     NOT NULL DEFAULT 1,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT events_end_after_start CHECK (ends_at > starts_at)
);

CREATE TRIGGER events_set_updated_at
    BEFORE UPDATE ON events
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE rooms (
    id         uuid        PRIMARY KEY DEFAULT uuidv7(),
    event_id   uuid        NOT NULL REFERENCES events (id) ON DELETE CASCADE,
    name       text        NOT NULL,
    capacity   integer     NOT NULL CHECK (capacity > 0),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (event_id, name)
);

CREATE TRIGGER rooms_set_updated_at
    BEFORE UPDATE ON rooms
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- The role sits on the membership row, not on the user, so there is no column anywhere
-- that could express a global role.
CREATE TABLE event_members (
    event_id   uuid        NOT NULL REFERENCES events (id) ON DELETE CASCADE,
    user_id    text        NOT NULL REFERENCES auth.users (id) ON DELETE CASCADE,
    role       text        NOT NULL REFERENCES roles (name),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (event_id, user_id)
);

-- For listing the events one user belongs to; the other direction is the primary key.
CREATE INDEX event_members_user_id_idx ON event_members (user_id);

CREATE TRIGGER event_members_set_updated_at
    BEFORE UPDATE ON event_members
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- room_id is nullable so a session can exist before anyone decides where it happens.
-- version is optimistic concurrency: an update carries the version it was read at, so
-- the second of two people editing the same session is told instead of overwriting.
CREATE TABLE sessions (
    id          uuid        PRIMARY KEY DEFAULT uuidv7(),
    event_id    uuid        NOT NULL REFERENCES events (id) ON DELETE CASCADE,
    room_id     uuid        REFERENCES rooms (id) ON DELETE SET NULL,
    title       text        NOT NULL,
    description text        NOT NULL DEFAULT '',
    starts_at   timestamptz NOT NULL,
    ends_at     timestamptz NOT NULL,
    version     integer     NOT NULL DEFAULT 1,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT sessions_end_after_start CHECK (ends_at > starts_at)
);

-- No double-booking, enforced here rather than by a SELECT-then-INSERT in a handler, so
-- two writes racing cannot both win.
--
-- '[)' means a session ending at 10:00 and one starting at 10:00 do not overlap. The
-- WHERE clause keeps unscheduled sessions out: two with no room are not in conflict.
--
-- Postgres 18's WITHOUT OVERLAPS is shorter but needs a stored range column and cannot
-- be partial, which would force every session to have a room. See ADR.
ALTER TABLE sessions
    ADD CONSTRAINT sessions_room_not_double_booked
    EXCLUDE USING gist (
        room_id WITH =,
        tstzrange(starts_at, ends_at, '[)') WITH &&
    ) WHERE (room_id IS NOT NULL);

-- Serves the schedule view: one event's sessions across a week, in order.
CREATE INDEX sessions_event_id_starts_at_idx ON sessions (event_id, starts_at);

CREATE TRIGGER sessions_set_updated_at
    BEFORE UPDATE ON sessions
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE session_speakers (
    session_id uuid        NOT NULL REFERENCES sessions (id) ON DELETE CASCADE,
    user_id    text        NOT NULL REFERENCES auth.users (id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (session_id, user_id)
);

CREATE INDEX session_speakers_user_id_idx ON session_speakers (user_id);

CREATE TABLE invitation_statuses (
    name text PRIMARY KEY
);

INSERT INTO invitation_statuses (name) VALUES
    ('pending'), ('accepted'), ('declined'), ('revoked');

-- Invited by email, because the person may not have an account yet. user_id is filled
-- in on acceptance.
CREATE TABLE invitations (
    id         uuid        PRIMARY KEY DEFAULT uuidv7(),
    event_id   uuid        NOT NULL REFERENCES events (id) ON DELETE CASCADE,
    email      text        NOT NULL,
    role       text        NOT NULL REFERENCES roles (name),
    status     text        NOT NULL DEFAULT 'pending' REFERENCES invitation_statuses (name),
    invited_by text        REFERENCES auth.users (id) ON DELETE SET NULL,
    user_id    text        REFERENCES auth.users (id) ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (event_id, email)
);

-- uuidv7 keys sort by creation time, so this one index is both the keyset cursor and
-- the chronological sort. That is why the primary keys are uuidv7 and not random.
CREATE INDEX invitations_event_id_id_idx ON invitations (event_id, id);
CREATE INDEX invitations_email_idx ON invitations (email);

CREATE TRIGGER invitations_set_updated_at
    BEFORE UPDATE ON invitations
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- +goose Down

DROP TABLE invitations;
DROP TABLE invitation_statuses;
DROP TABLE session_speakers;
DROP TABLE sessions;
DROP TABLE event_members;
DROP TABLE rooms;
DROP TABLE events;
DROP FUNCTION set_updated_at();
DROP TABLE time_zones;
