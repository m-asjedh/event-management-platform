# Event Management Platform

Events — conferences, weddings, gatherings — with sessions scheduled inside them, rooms, and people.

Roles are per event, not global: the same user can be an admin on one event and an attendee on another.

## Stack

Go · PostgreSQL 18 · OpenAPI · React + Vite · TypeScript

## Where to find things

- Per-event authorization + response filtering: `backend/internal/authz/`
  (`present_test.go` proves an attendee sees no roster/emails)
- Room conflict + version check: `backend/internal/sessions/` (`patch_test.go`)
- Keyset pagination: `backend/internal/invitations/`
- DST / time handling: `backend/internal/tz/` (`tz_test.go`)
- Contract CI (server honours the spec): `backend/internal/contract/`
- ADRs: `docs/adr/` — the three that count are 0001, 0002, 0006
- AI workflow write-up: `AI-WORKFLOW.md` · Tradeoffs: `TRADEOFFS.md`

## Running it

Reviewers verify the build with this. It does **not** start the UI.

```bash
make up && make seed && make test
```

Requires Docker, Docker Compose and make. Nothing else.

`make seed` is the fast default. It COPY-loads a handful of events (DST spring/fall,
the prompt-injection fixture, several IANA zones), 80 users, and 400 invitations —
enough to page and to sign in as `seed.admin@` / `seed.attendee@`. `make seed-full`
is the same generator with the volume turned up: 50 events, 5k users, 50k invitations,
for the keyset/pagination demo.

Tests run against the small seed. Walk-to-end and mid-traversal insert neither
repeat nor skip at any size. The keyset index plan is proven in-test: 50 rows is
not enough for the planner to pick `invitations_event_id_id_idx`, so that test
inserts its own throwaway event. The captured 50k EXPLAIN is still in
`docs/postgres-18.md` (`make seed-full`).

All seed accounts share one real scrypt hash to keep seeding fast; not a production pattern. Demo password: `correct-horse-battery`. Either target only talks to the local Compose database and truncates first so it can rerun.

To **see the UI**, start the stack and then the frontend (a third command on purpose — `make up` stays the fast backend path):

```bash
make up && make seed && make frontend
```

Then open http://localhost:5173. `GET /healthz → ok` means the typed call reached the API.

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

## Frontend

Vite, TanStack Router, TanStack Query, Tailwind, shadcn/ui. Types are generated from `openapi/openapi.yaml`.

The UI is not part of `make up`. After the stack is up (see **Running it**):

```bash
make frontend          # http://localhost:5173
```

Sign in as `seed.admin@example.com` / `correct-horse-battery`. The home page lists events you can read; each one opens `/events/$eventId/schedule`. Use the Sign out control to switch accounts (e.g. `seed.admin@` → `seed.attendee@`) — no cookie clearing needed.

That schedule shows rooms across and time down, in the event's IANA zone. Drag a session to a new
room or time. A rejected write rolls the block back (or to `currentState` on `STALE_VERSION`); the
grid always matches the server.

Invitations are at `/events/$eventId/invitations`: keyset pages, virtualized rows, fetch on
scroll. `?cursor=` is the opaque keyset token; `?status=` filters loaded rows in the client.

The agent UI is `/agent`. Reads use the public GET API as you. Writes pause on an approval card
that shows the exact method, path, and JSON; Approve is the only code path that sends that body.

```bash
make gen-api           # regenerate types after a spec change
make check-generated   # CI: fail if committed types are stale
make frontend-build    # tsc + vite build
make frontend-test     # Vitest: schedule, drag recovery, invitations, agent gate, URL state
```
