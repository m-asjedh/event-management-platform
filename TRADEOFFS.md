# Tradeoffs

What I chose and what I deliberately skipped. A cut that is written down is a
decision; a silent one looks like I forgot. This reflects the finished
submission — backend, the Track 3 frontend, and both agents.

---

## Which track, and why

**Track 3 — Fullstack.**

The interesting problem is the seam: the client moves a session, the server
says no, and the UI has to recover without lying. Backend-only would prove the
constraint and stop. I picked Fullstack because that recovery has to be real —
and it is: drag-to-reschedule applies optimistically, then reconciles to server
truth on ROOM_CONFLICT / STALE_VERSION. The database and authz edge came first
so the UI always had something truthful to talk to.

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

**Optimistic concurrency.** `sessions.version` is checked in the same UPDATE
that increments it (`PATCH /sessions/{id}`). A mismatch is STALE_VERSION, and
the error body carries `currentState` so the client reconciles without a second
GET. The drag UI uses exactly that.

**Errors on the routes that exist.** `{code, reason}` JSON, not `text/plain`.

**50k invitations.** Keyset list on `(event_id, id)` because uuidv7 is already
time-ordered; `EXPLAIN` is in `docs/postgres-18.md`. The endpoint, the opaque
cursor, and a virtualized fetch-on-scroll UI are all built; a mid-traversal
insert neither repeats nor skips a row (tested). Default seed is small per
reviewer guidance (confirmed with Prajwal by email); `make seed-full` loads the
50k. Scale tests assert invariants at any size; the at-scale EXPLAIN plan is
captured under `seed-full`.

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

**Read-only CLI agent.** A bounded GET loop over the public API, with the user's Better Auth cookie.
No database, no extra key, no hosted model — a clean machine has no API key, and the three
scenarios have to be deterministic in `make test`.

**Agent UI.** The write-capable loop runs in the browser as the signed-in user (same cookie, same
public routes). Streaming is React state: each GET, proposal, and result is pushed as it happens.
I did not add an agent SSE endpoint or run tables — approve/deny/interrupt stay in the same
process as the loop, so the client/server line is the public API. Reload does not restore a pending
approval; say that if a reviewer asks.

The gate is `executeApprovedWrite` in `frontend/src/lib/agent/gate.ts`. The planner cannot call it.
Approve mints a ticket bound to the frozen JSON; only then does `sendWrite` run. Deny and interrupt
never mint a ticket.

Injection: the seeded description is scanned on every GET. A warning is shown. There is no
role-grant tool and no such route in the spec. Even if a fooled planner proposes
`POST /events/{id}/members`, the allowlist refuses to send it. The layer I rely on is still the
**API session** (the request is the user's, `Grant.Can` is unchanged). The code gate and the missing
route hold even if the model obeys the injected text.

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

In rough priority order — each builds on what already exists:

1. **Audit log, wired.** The append-only who/what/before-after table is cut
   today because nothing writes it. Two weeks buys the write path on every
   mutation (create/patch/room), keyed off the same `Grant` context, plus a
   read endpoint. The schema shape is known; the work is the write hook and the
   query.

2. **Server-side invitation filtering.** `status` is a client-side filter of
   already-loaded pages today, because `GET /events/{id}/invitations` has no
   status param. Two weeks buys a real server filter that composes with the
   keyset cursor (filter + cursor in one indexed query), so a shared
   `?status=` link is correct across the full 50k, not just loaded rows.

3. **Speaker-clash detection in the schedule.** DB-enforced speaker
   double-booking was cut (an exclusion can't span two tables). Two weeks buys
   the query-time detection the schedule view was designed for: return speaker
   assignments in `GET /sessions` and flag overlaps client-side, the same way
   room clashes are shown now — detection without a hard constraint.

4. **Server-side agent run.** The write-capable loop runs in the browser, so a
   reload loses a pending approval. Two weeks buys a server-side run with an SSE
   stream and a small run table, so approvals survive reload, multiple clients
   can observe a run, and the sequence of calls is reconstructable from the
   server, not just React state.

5. **Shared `23P01` → ROOM_CONFLICT helper.** `Apply` (PATCH) and `Insert`
   (POST) each translate the exclusion violation today. Small, deliberate
   duplication; two weeks folds it into one mapping so the error shape can never
   drift between the two write paths.

6. **Recurring sessions.** Store as rules with exceptions plus schedule history,
   rather than materialized rows. This is the one item that needs real design,
   not just wiring — it changes how the schedule is queried and how a single
   occurrence is edited.
