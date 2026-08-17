# ADR 0002 — Room Conflicts and Session Version

## Decision

PostgreSQL is the authority for room double-booking. `sessions` has a partial GiST exclusion constraint:

```sql
EXCLUDE USING gist (
    room_id WITH =,
    tstzrange(starts_at, ends_at, '[)') WITH &&
) WHERE (room_id IS NOT NULL)
```

`'[)'` means a session ending at 10:00 and one starting at 10:00 do not overlap. Sessions with no room are left out, so two drafts are not in conflict.

Speaker double-booking is not a database constraint. An exclusion cannot span `sessions` and `session_speakers` without copying times onto the join table and keeping them in sync with a trigger. That is a cut, not an accident.

Concurrent edits use a `version` integer on `sessions`. An update is meant to succeed only when the supplied version matches.

## Rejected Alternatives

**Check if the room is free, then insert**

Two requests can both see the room as free. A race would let both win. The constraint is what stays correct when two writes happen at the same instant.

**PostgreSQL 18 `UNIQUE (room_id, during WITHOUT OVERLAPS)`**

Shorter, but it needs a stored range column and cannot be partial. Every session would then need a room. Drafts without a room are part of the schema, so the GiST exclusion with a `WHERE` clause stayed.

**Last-write-wins**

The second save would silently overwrite the first person's edit.

## Why

The room constraint holds even when two writes race. The `version` column is what stops two people editing the same session from silently clobbering each other.
