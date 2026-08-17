# Tradeoffs

Honesty about what I chose, what I skipped on purpose, and what more time would buy. A cut that is
written down is a decision. A cut that is silent looks like I forgot.

This file is current as of the work that exists today. I will add a line when I skip something later.

---

## Which track, and why

**Track 3 — Fullstack.**

The interesting problem in this domain is the seam: the client moves a session, the server says no,
and the UI has to recover without lying about the schedule. Backend-only would prove the constraint
and stop. Applied AI would prove the agent and stop. Fullstack is the track where both the room
conflict and the stale-version case have to become something a person can see and undo.

---

## Decisions this track forced

These are decided even where the UI is not built yet. The database and the edge have to be right
first, or the frontend has nothing truthful to react to.

**Auth on a clean machine.** The assignment wants off-the-shelf auth and `make up` with Docker only.
Auth0 needs an account the reviewer does not have. Better Auth runs in Compose against tables the
migrations own.

**Authorization.** Roles are per event. One chokepoint: `Grant.Can(permission)`. The same check
filters the body (`PresentEvent`), so an attendee GET does not leak the roster or emails.
`TestAttendeeEventOmitsRosterAndEmails` and `TestGrantDoesNotSwitchOnRoleName` are the proof.

**"The room is taken."** PostgreSQL holds a partial GiST exclusion on `(room_id, tstzrange)`.
Check-then-insert loses a race. PostgreSQL 18 `WITHOUT OVERLAPS` was the obvious 18-native form; it
needs a stored range and cannot be partial, so every session would need a room. Drafts without a
room are part of the domain, so GiST with `WHERE room_id IS NOT NULL` stayed.

**"Someone else edited it first."** `sessions.version` is in the schema. The update that returns
`STALE_VERSION` is not wired yet. The column is there so that handler has something to check.

**Errors the client can branch on.** Failures are `{code, reason}` JSON, not `text/plain`. The
schedule will need `ROOM_CONFLICT` and `STALE_VERSION` as separate codes; both are 409.

**50k invitations.** Seeded. Keyset is `(event_id, id)` because uuidv7 is already time-ordered.
`EXPLAIN` on the seeded table is in `docs/postgres-18.md`. The list endpoint is not built yet.

**Time.** Stored as `timestamptz` plus `events.time_zone`. Local input goes through `tz.Instant`,
which refuses a spring-forward gap and a fall-back fold. Display uses the event's zone, proven under
`TZ=Pacific/Auckland`.

**Where fetching will live (not built).** TanStack Router owns URL and position (so a shared link
can reproduce a cursor). TanStack Query owns cache, optimistic updates, and rollback. That split is
the answer to "where does fetching live"; the app is next.

---

## What I cut, and why

**Speaker double-booking in the database.** An exclusion cannot span `sessions` and
`session_speakers`. Doing it for real means copying times onto the join table and a trigger to keep
them in sync. Track 3 needs one real `"the room is taken"` signal. Rooms give that. Speakers can
still be flagged in the schedule query later. Claiming the database forbids speaker clashes would be
the dishonest version of this cut.

**A `created_at` column used only to page invitations.** uuidv7 already sorts by creation time.
`(event_id, id)` is the cursor and the order. A wider `(event_id, created_at, id)` index would work
and would also duplicate the key.

**PostgreSQL 18 `WITHOUT OVERLAPS` for rooms.** Shorter syntax. It cannot be a partial unique, so
unscheduled sessions would be illegal. The older GiST exclusion with a `WHERE` clause is the one
that matches the domain.

**One scrypt hash for all 5,000 seed accounts.** A real Better Auth hash (`N=16384, r=16, p=1`).
Computing it 5,000 times would dominate `make seed` for no extra proof. Demo password:
`correct-horse-battery`. Not a production pattern.

**Auth0 / Ory.** Self-hosted Better Auth is what makes `make up` work offline.

**Audit log table, session-dependency table, bulk-invite idempotency key.** Track 1 items, or
tables with nothing to write them. An unused column looks like abandoned work. Left out on purpose.

---

## What two more weeks would buy

In order, because this is the rest of the track, not a wish list:

- Session create/update that parse local times through `tz.Instant` and return `ROOM_CONFLICT` /
  `STALE_VERSION` with enough state for the client to recover.
- Invitation keyset list on the 50k seed, opaque cursor in the URL.
- TanStack schedule: optimistic drag, rollback that branches on `code`.
- Speaker clashes visible in the schedule query (the cut above, shown rather than enforced).
- HITL agent UI, writes only after approval, attack text in the seeded event as the injection case.
- OpenAPI codegen and a CI job that sends real requests at the running server.
- ADR 0006 in `docs/adr/` once the data-loading split is running code, not only a decision.
