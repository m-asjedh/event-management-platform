-- Authorization as data. The grant matrix is a table the code reads at request time,
-- so adding a role is an INSERT, not a handler change.
--
-- Permissions are finer than the endpoints on purpose: `member.read` and
-- `user.email.read` let the same check also decide what stays out of the response body.

-- +goose Up

CREATE TABLE roles (
    name        text PRIMARY KEY,
    description text NOT NULL
);

CREATE TABLE permissions (
    name        text PRIMARY KEY,
    description text NOT NULL
);

CREATE TABLE role_permissions (
    role       text NOT NULL REFERENCES roles (name) ON DELETE CASCADE,
    permission text NOT NULL REFERENCES permissions (name) ON DELETE CASCADE,
    PRIMARY KEY (role, permission)
);

INSERT INTO roles (name, description) VALUES
    ('admin',       'Full control of the event, including membership and roles'),
    ('contributor', 'Manages sessions and invites attendees, cannot change roles'),
    ('attendee',    'Reads the event and its sessions, nothing more');

INSERT INTO permissions (name, description) VALUES
    ('event.read',         'See an event'),
    ('event.update',       'Change an event'),
    ('event.delete',       'Delete an event'),
    ('room.read',          'See the rooms of an event'),
    ('room.manage',        'Create, change and delete rooms'),
    ('session.read',       'See the sessions of an event'),
    ('session.create',     'Add a session'),
    ('session.update',     'Change or reschedule a session'),
    ('session.delete',     'Delete a session'),
    ('member.read',        'See who belongs to an event and with which role'),
    ('member.invite',      'Invite people to an event'),
    ('member.remove',      'Remove people from an event'),
    ('member.role.update', 'Change somebody''s role on an event'),
    ('invitation.read',    'See the invitations of an event'),
    ('invitation.create',  'Create invitations'),
    ('invitation.revoke',  'Revoke an invitation'),
    ('user.email.read',    'See email addresses in a response body');

-- admin holds everything, including permissions added by a later migration
INSERT INTO role_permissions (role, permission)
SELECT 'admin', name FROM permissions;

INSERT INTO role_permissions (role, permission) VALUES
    ('contributor', 'event.read'),
    ('contributor', 'room.read'),
    ('contributor', 'session.read'),
    ('contributor', 'session.create'),
    ('contributor', 'session.update'),
    ('contributor', 'session.delete'),
    ('contributor', 'member.read'),
    ('contributor', 'member.invite'),
    ('contributor', 'invitation.read'),
    ('contributor', 'invitation.create'),
    ('contributor', 'user.email.read');

-- No member.read, no user.email.read. The response filtering test asserts on this.
INSERT INTO role_permissions (role, permission) VALUES
    ('attendee', 'event.read'),
    ('attendee', 'room.read'),
    ('attendee', 'session.read');

-- +goose Down

DROP TABLE role_permissions;
DROP TABLE permissions;
DROP TABLE roles;
