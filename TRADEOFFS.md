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

**Event creation.** Gated on authentication only, since per-event roles cannot predate the event.
There is no `event.create` permission. The creator becomes admin of the new event.

**Existence hiding.** Nested event routes check the grant before the event row, same as
`GET /events/{id}`. A caller who is not a member gets `FORBIDDEN` whether the event exists or not.
404-before-grant would leak existence (404 vs 403).

**Room double-booking.** Partial GiST exclusion on `(room_id, tstzrange)`. Check-then-insert loses a
race. PostgreSQL 18 `WITHOUT OVERLAPS` cannot be partial, so it would force every session to have a
room. Drafts without a room stay legal.

**Optimistic concurrency (schema only).** `sessions.version` exists. No update handler uses it yet.

**Errors on the routes that exist.** `{code, reason}` JSON, not `text/plain`.

**50k invitations (seed + index, no list API).** Keyset index is `(event_id, id)` because uuidv7 is
already time-ordered. `EXPLAIN` is in `docs/postgres-18.md`.

**Time.** `timestamptz` plus `events.time_zone`. `tz.Instant` refuses a spring-forward gap and a
fall-back fold. Display uses the event's zone, proven under `TZ=Pacific/Auckland`.

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

**A shared helper for `23P01` → `ROOM_CONFLICT`.** `Apply` (PATCH) and `Insert` (POST) each translate
the exclusion violation. Left duplicated so the PATCH store is not rewritten this pass.

**Read-only agent.** A bounded GET loop over the public API, with the user's Better Auth cookie.
No database, no extra key, no hosted model — a clean machine has no API key, and the three
scenarios have to be deterministic in `make test`. Write access and a human-in-the-loop UI are later.

**Frontend types.** TypeScript types come from `openapi/openapi.yaml` via `openapi-typescript`.
Hand-written response interfaces are not allowed. `make check-generated` regenerates the file and
fails if git is dirty. The UI is not part of `make up` — a Vite pull would risk the 5-minute budget.

**Data loading.** TanStack Router owns where the user is: `/events/$eventId/schedule` and `?day=`
(ADR 0006). TanStack Query owns server data: the route loader calls `ensureQueryData` for event,
rooms, and sessions; the page reads the same query options with `useSuspenseQuery`. Drag uses a
Query mutation: optimistic cache, then `onError` rolls back (or applies `currentState` on
`STALE_VERSION`) without a refetch.

**Schedule layout.** Rooms as columns, time as rows, one selected day. Seed sessions sit on an
hourly room grid; a same-room clash is then two blocks in one column. A days-across week would hide
that unless every day was also split by room. Week navigation is the URL so a day is shareable.

**Event-zone display.** Instants are formatted with `Intl.DateTimeFormat` and the event's IANA name
from `GET /events/{id}`. The browser zone is never used. Proven under `TZ=Pacific/Auckland`.
PATCH sends the same event-local wall clock the user saw (`2026-03-08T10:00:00`, no offset).

**Conflicts.** Same-room overlaps are computed client-side from the fetched sessions (`[start, end)`).
The database still rejects room double-booking on write; the view has to be able to *show* a clash
(speaker overlaps are not in `GET /sessions`, so they cannot be marked).

**Drag.** `@dnd-kit/core` (pointer) on `SessionBlock`. Native HTML5 DnD is a poor fit for a
positioned grid and for tests. Drop computes room + snapped time; the mutation is the source of
truth, not the drag library.

**Invitations list.** `useInfiniteQuery` pages on the opaque `nextCursor` (never parsed, never
an offset). `@tanstack/react-virtual` keeps only the visible rows in the DOM. Fetch-on-scroll
loads the next page; the last page (`nextCursor` absent) stops cleanly. Position in the URL is
that same opaque cursor (`replace`, not a pixel or page number), so a shared link starts the
keyset at that slice. `status` is a client-side filter of already-loaded pages: `GET
/events/{id}/invitations` has no status query param, and the spec is not changing.

---

## What two more weeks would buy

The next work that uses what is already here: the write-capable agent UI with an approval gate.
I am not pre-writing that here.
