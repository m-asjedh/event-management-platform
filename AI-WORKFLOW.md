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

**I drove**

- Time at the edge: a local wall clock must map to a unique instant, or the write is refused. Go's
  `time.Date` is not that mapping. Session create/update must go through `tz.Instant` when those
  handlers exist — the helper only protects writes that actually call it.

**I delegated**

- The first pass at "write the DST tests" looked like testing existing conversion. There was no
  conversion package. The model reached for `time.Date`. That is the call I rejected, not the tests.

---

## How I planned

For the DST tests: what must be true (a civil time is one instant, or it is not a time we accept),
where that is enforced (the edge, `internal/tz`, not the database and not the test), what the caller
gets (`ErrNonexistentLocalTime` / `ErrAmbiguousLocalTime`, not a nearby instant), and which test
proves each case. The tests call `Instant` and `WallClock`. They do not reimplement conversion.

---

## Where the AI produced wrong code

Rules for this section: paste the actual code, not a description of it. Say why it looked correct — if
it looked obviously wrong it is not worth recording. Say how I noticed. Name the test that catches it
now.

### `time.Date` as a local-time parser

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
- `TestCrossZoneLocalRendering` — seeded `Conference 10` (`Asia/Colombo`) is `09:00 +05:30`, not UTC
  `03:30`, not Auckland `15:30`. `make test` sets `TZ=Pacific/Auckland` so that last check is real.

---

## Log


| Date       | Notes                                                                              |
| ---------- | ---------------------------------------------------------------------------------- |
| 2026-08-17 | git-ai installed and verified before the first commit. Repo created, notes pushed. |
| 2026-08-17 | Seed via COPY (~800ms, 50k invitations). Shared scrypt hash is a seed-speed choice. uuidv7 minted in Go so COPY has parent IDs and reruns match. Invitations keyset EXPLAIN captured in docs/postgres-18.md. |
| 2026-08-17 | DST edge: `time.Date` turns 02:30 on 2026-03-08 NY into 01:30 EST with no error. `internal/tz.Instant` refuses gaps and folds. Three tests in `tz_test.go`. Seed rows were fine (09:00–16:00). Session writes must use `Instant` when those handlers exist. |
