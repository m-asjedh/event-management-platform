# Tradeoffs

What I have chosen and skipped in the work that exists today. I will add a line when I skip
something later. I am not listing the rest of the track here.

---

## Which track, and why

**Track 3 — Fullstack.**

The interesting problem is the seam: the client moves a session, the server says no, and the UI has
to recover without lying. Backend-only would prove the constraint and stop. I picked Fullstack
because that recovery has to be real. The UI is not built yet. The database and the authz edge are,
so that UI has something truthful to talk to.

---

## Decisions already in the repo

**Auth on a clean machine.** Better Auth in Compose, tables owned by migrations. Auth0 would need an
account the reviewer does not have.

**Authorization.** Roles are per event. `Grant.Can(permission)` is the chokepoint. `PresentEvent`
drops the roster and emails unless those permissions are granted.
`TestAttendeeEventOmitsRosterAndEmails` and `TestGrantDoesNotSwitchOnRoleName` are the proof.

**Room double-booking.** Partial GiST exclusion on `(room_id, tstzrange)`. Check-then-insert loses a
race. PostgreSQL 18 `WITHOUT OVERLAPS` cannot be partial, so it would force every session to have a
room. Drafts without a room stay legal.

**Optimistic concurrency (schema only).** `sessions.version` exists. No update handler uses it yet.

**Errors on the routes that exist.** `{code, reason}` JSON, not `text/plain`.

**50k invitations (seed + index, no list API).** Keyset index is `(event_id, id)` because uuidv7 is
already time-ordered. `EXPLAIN` is in `docs/postgres-18.md`.

**Time.** `timestamptz` plus `events.time_zone`. `tz.Instant` refuses a spring-forward gap and a
fall-back fold. Display uses the event's zone, proven under `TZ=Pacific/Auckland`.

OpenAPI covers `GET /healthz`, `GET /me`, and `GET /events/{id}` only.

---

## What I cut, and why

**Speaker double-booking in the database.** An exclusion cannot span `sessions` and
`session_speakers` without copying times onto the join table. Rooms already give a real constraint.
Claiming speakers cannot clash would be a lie.

**A `created_at` column used only to page invitations.** uuidv7 already sorts by creation time.
`(event_id, id)` is enough.

**PostgreSQL 18 `WITHOUT OVERLAPS` for rooms.** Cannot be partial. Unscheduled sessions would become
illegal.

**One scrypt hash for all 5,000 seed accounts.** A real Better Auth hash, computed once, so `make
seed` stays fast. Demo password: `correct-horse-battery`. Not a production pattern.

**Auth0 / Ory.** Self-hosted Better Auth is what makes `make up` work offline.

**Audit log table, session-dependency table, bulk-invite idempotency key.** Nothing writes them
today. An unused column looks like abandoned work.

---

## What two more weeks would buy

The next work that uses what is already here: session writes through `tz.Instant` and the room
constraint, and an invitation list on the index we already measured. I am not pre-writing the
frontend, the agent, or CI in this file.
