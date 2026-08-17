# Event Management Platform

Events — conferences, weddings, gatherings — with sessions scheduled inside them, rooms, and people.

Roles are per event, not global: the same user can be an admin on one event and an attendee on another.

## Stack

Go · PostgreSQL 18 · OpenAPI · React + Vite · TypeScript

## Running it

```bash
make up && make seed && make test
```

Requires Docker, Docker Compose and make. Nothing else.

`make seed` only talks to the local Compose database. It truncates first so it can rerun. All seed accounts share one real scrypt hash to keep seeding fast; not a production pattern. Demo password: `correct-horse-battery`.

## Read-only agent

A small CLI that answers questions by calling the public HTTP API as a signed-in user. GET only. No database access, no model key.

```bash
make up && make seed
make agent-scenarios
```

Or one question:

```bash
QUESTION='Which events are in America/New_York?' make agent
```

Interactive (signed in as `seed.admin@example.com` unless `AGENT_EMAIL` is set):

```bash
make agent
```

The three scripted scenarios:

1. As `seed.admin@example.com`: "Which events are in America/New_York?" → `GET /events`, then filter on `timeZone`.
2. As `seed.admin@example.com`: "How many sessions does DST Spring Forward have?" → `GET /events`, then `GET /events/{id}/sessions`, then count.
3. As `seed.attendee@example.com`: "Is seed.attendee allowed to see the invitations for Prompt Injection Conference?" → `GET /events`, then `GET /events/{id}/invitations`. The API returns 403; the agent reports the denial and does not invent a list.

`make test` runs those three against an in-process fake HTTP API (no live model, no API key). `make agent-scenarios` runs them against the real seeded server.

_Under construction._
