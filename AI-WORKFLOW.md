# AI Workflow

Written as the work happens, not reconstructed at the end.

---

## Tools


| Tool   | Model    | Used for                                                                    |
| ------ | -------- | --------------------------------------------------------------------------- |
| Claude | Opus 5   | Reading the requirements, planning, reviewing decisions before writing code |
| Cursor | Grok 4.6 | Writing code                                                                |
| git-ai | 1.6.22   | Line-level AI/human attribution, in `refs/notes/ai` from the first commit   |


```bash
git log --show-notes=ai   # attribution per commit
```

---

## What I drove vs. what I delegated

What exists today: the full backend (auth, per-event authz with body filtering,
room exclusion + version check, keyset pagination, DST edge, 50k seed, contract
CI, a read-only CLI agent) and the full Track 3 frontend (schedule, drag +
typed recovery, virtualized invitations, typed URL state, and the
write-capable agent with an out-of-model approval gate).

**I drove**

- Authorization as one chokepoint (`Grant.Can`) that also filters the response body. No `if role ==`.
- Room double-booking in PostgreSQL. Check-then-insert is a race.
- Speaker double-booking is not a database constraint. That is a cut, not a missing feature.
- Domain keys are uuidv7. The invitations index is `(event_id, id)`, not `(created_at, id)`.
- Times are `timestamptz` plus the event's IANA zone. `tz.Instant` is the parse edge; `time.Date` is not.
- Auth is Better Auth inside Compose, so `make up` does not need an external account.

**I delegated**

- Compose, Dockerfiles, Makefile, first-draft SQL, handler and store boilerplate.
- The seed's COPY loop, once the counts, fixtures, and "uuidv7 in Go" rule were fixed.
- First drafts of the ADRs and of the DST tests. Both needed catching (below).
- First-draft frontend (shell, queries, drag, agent panel). Types, the gate, and the Me alias
  had to be checked against the generated-types rule.

---

## How I planned

Same rule throughout, backend and frontend: decide what must stay true (a grant,
a room, a unique instant, "the grid never lies about server state"), put it in
one place — the database, one function, or one gate — then write a test that
calls that place, not a copy of the logic. On the frontend that meant asserting
on API call paths and post-rejection state, not on model wording or pixels.

---

## Where the AI produced wrong code

Rules for this section: paste the actual code, not a description of it. Say why it looked correct — if
it looked obviously wrong it is not worth recording. Say how I noticed. Name the test that catches it
now.

### 1. ADRs that put back decisions I had already rejected

The first ADR / architecture drafts looked finished. They compiled as prose. They quietly put back
two things I had already said no to.

**Speaker constraint.** I had already cut database-enforced speaker clashes: an exclusion cannot
span `sessions` and `session_speakers` without copying times onto the join table and keeping them in
sync. The draft put the copy back:

```sql
ALTER TABLE session_speakers
    ADD COLUMN during TSTZRANGE NOT NULL,
    ADD CONSTRAINT session_speakers_no_overlap
        UNIQUE (user_id, during WITHOUT OVERLAPS);
```

That looks correct. It is the PostgreSQL 18 form. It would even work — until a session is moved and
the copied `during` is not. Then the constraint guards the old time and says nothing.

**Extra timestamp for paging.** I had already chosen uuidv7 so `id` *is* the order. The draft still
keyed the invitation list on `created_at`:

```sql
CREATE INDEX idx_invitations_keyset
    ON invitations (event_id, created_at DESC, id DESC);
```

That looks correct. It works. It also duplicates order that the primary key already has, which is
the alternative ADR 0004 rejects.

I noticed both by reading the draft against the decisions I had already written, not by waiting for
a test to fail. What shipped instead: `session_speakers` has no `during` column; ADR 0002 records
the speaker cut; the index is `invitations_event_id_id_idx ON invitations (event_id, id)`.

There is no test that "catches an ADR". The proof it did not land is `backend/migrations/00003_domain.sql`.

### 2. `time.Date` as a local-time parser

The task was: prove time-zone handling. The obvious next step was "test the conversion we already
have." There was no conversion. The seeder (and the stdlib call anyone reaches for) is this:

```go
st := time.Date(sessionDay.Year(), sessionDay.Month(), sessionDay.Day(), hour, 0, 0, 0, loc)
```

It looks correct. It takes a zone. It returns `time.Time`. It does not return an error. On the hours
the seeder actually uses (09:00–16:00) it *is* correct, including on the DST fixture days. The stored
rows were never wrong. This was a latent bug: it appears the moment a user types an early-morning
wall clock.

I ran the same call on the gap the tests require:

```go
ny, _ := time.LoadLocation("America/New_York")
got := time.Date(2026, time.March, 8, 2, 30, 0, 0, ny)
// got.Format(time.RFC3339) == "2026-03-08T01:30:00-05:00"
```

02:30 never happens that morning (clocks jump 02:00 → 03:00). `time.Date` does not say so. On Go 1.26
it returns **01:30 EST** — a real instant, silently, the wrong hour. Wrapping that in a helper and
asserting "no error" would have certified the footgun.

What I did instead: `internal/tz`. `Instant` refuses a gap (`ErrNonexistentLocalTime`) and a fold
(`ErrAmbiguousLocalTime`). `Occurrences` returns both fall-back instants so they are not collapsed.
`WallClock` renders in the event's IANA zone.

Caught by:

- `TestDSTSpringForwardNonexistentLocalTime` — `Instant(..., 2, 30, 0)` is `ErrNonexistentLocalTime`,
  not 01:30 EST and not 03:30 EDT.
- `TestDSTFallBackAmbiguousLocalTime` — `Instant(..., 1, 30, 0)` is `ErrAmbiguousLocalTime`; the two
  01:30s are `2026-11-01T01:30:00-04:00` and `2026-11-01T01:30:00-05:00`.
- `TestCrossZoneLocalRendering` — the seeded Asia/Colombo event is `09:00 +05:30`, not UTC
  `03:30`, not Auckland `15:30`. `make test` sets `TZ=Pacific/Auckland` so that last check is real.

### 3. A generated type that doesn't exist by that name

Wiring sign-out needed a `Me` type. The agent reached for the obvious form:

```ts
export type Me = components["schemas"]["Me"]
```

This looks correct — it's how you alias a named schema, and a `Me` schema does
exist in the spec. But the rest of the codebase derives response types from the
path, not the schema name (see `SessionList`, `RoomList`, `PatchedSession`).
Mixing the two styles is the kind of thing that compiles until a spec refactor
renames or inlines a schema and every schema-name alias breaks at once.

I kept the path form, consistent with the existing types and independent of
whether the shape is a named schema:

```ts
export type Me =
  paths["/me"]["get"]["responses"]["200"]["content"]["application/json"]
```

Caught by reading it against the existing alias pattern, not by a failing build
(the schema-name form happened to compile here). Both `paths[...]` and
`components["schemas"][...]` are valid spec-derived aliases used in
`frontend/src/lib/api/types.ts`; the point isn't which one — it's that the type
is derived from the generated schema, never hand-written.

---

## Where the AI surfaced a decision

Not a bug. Nested create/list first loaded the event (404 if missing) and then asked `Grant.Can`.
`GET /events/{id}` already did the opposite: grant first, so a missing event and a forbidden event
both look like `FORBIDDEN`.

404-before-grant tells an unauthorized caller whether the event exists (404 vs 403). That is an
information leak. It looked correct as REST ("the parent must exist"), which is why it is worth
recording: the safer rule is the one `Show` already had.

I aligned the nested routes to grant-first. `TestUnknownEventIsForbidden` asserts a well-formed
unknown event id is `FORBIDDEN`, not `NOT_FOUND`.

---

## Log


| Date       | Notes                                                                              |
| ---------- | ---------------------------------------------------------------------------------- |
| 2026-08-17 | git-ai installed and verified before the first commit. Repo created, notes pushed. |
| 2026-08-17 | Seed via COPY (~800ms, 50k invitations). Shared scrypt hash is a seed-speed choice. uuidv7 minted in Go so COPY has parent IDs and reruns match. Invitations keyset EXPLAIN captured in docs/postgres-18.md. |
| 2026-08-17 | DST edge: `time.Date` turns 02:30 on 2026-03-08 NY into 01:30 EST with no error. `internal/tz.Instant` refuses gaps and folds. Three tests in `tz_test.go`. Seed rows were fine (09:00–16:00). |
| 2026-08-17 | AI-WORKFLOW: ADR drafts had put back the speaker exclusion and a `created_at` keyset index. TRADEOFFS.md is the cuts that already exist, not the rest of the track. |
| 2026-08-17 | Nested CRUD listed events before the grant (404 vs 403 leaked existence). Aligned to `Show`: grant first. Event create stays auth-only; no `event.create` row. |
| 2026-08-18 | Track 3 frontend built: schedule (event-zone times, conflict flags), drag-to-reschedule with typed ROOM_CONFLICT/STALE_VERSION recovery, virtualized invitations over the keyset cursor, typed URL state, write-capable agent with an out-of-model approval gate. Types generated from the spec; `make check-generated` guards drift. |
| 2026-08-18 | Agent reached for `components["schemas"]["Me"]`; kept the `paths[...]` form to match the existing type-derivation pattern. Added a real Sign out (Better Auth sign-out) so admin↔attendee switching needs no cookie deletion. |
