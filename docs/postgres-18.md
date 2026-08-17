# PostgreSQL 18

The assignment asked for version 18. This is what that version is doing here, and why.

## `uuidv7()` as the domain primary key

PostgreSQL 18 can generate time-ordered UUIDs in the database. Domain tables use `uuid DEFAULT uuidv7()`.

uuidv7 values sort by creation time, so invitations need one index that is both the chronological order and the keyset cursor:

```sql
CREATE INDEX invitations_event_id_id_idx ON invitations (event_id, id);
```

Random UUIDs would not give that order. A separate `created_at` column would work, but it would duplicate order already present in the key. That choice is [ADR 0004](adr/0004-uuidv7-keys.md).

The seeder still mints uuidv7 values in Go. COPY has to know parent IDs before it inserts children, and a database default would make reruns non-reproducible.

## Keyset plan on the 50k seed

Captured 2026-08-17. 50,000 invitation rows. One event has 1,000 of them. Both pages use `invitations_event_id_id_idx` and stop after 50 rows.

First page (`WHERE event_id = $1 ORDER BY event_id, id LIMIT 50`):

```
Limit  (cost=0.41..59.40 rows=50 width=76) (actual time=0.008..0.015 rows=50.00 loops=1)
  Buffers: shared hit=5
  ->  Index Scan using invitations_event_id_id_idx on invitations
        Index Cond: (event_id = '…'::uuid)
        Index Searches: 1
        Buffers: shared hit=5
Planning Time: 0.024 ms
Execution Time: 0.022 ms
```

Keyset page (`WHERE event_id = $1 AND id > $2 ORDER BY event_id, id LIMIT 50`):

```
Limit  (cost=0.41..64.13 rows=50 width=76) (actual time=0.042..0.054 rows=50.00 loops=1)
  Buffers: shared hit=5
  ->  Index Scan using invitations_event_id_id_idx on invitations
        Index Cond: ((event_id = '…'::uuid) AND (id > '…'::uuid))
        Index Searches: 1
        Buffers: shared hit=5
Planning Time: 0.242 ms
Execution Time: 0.075 ms
```

## Other Postgres features in use (not 18-only)

These work on older versions. They are here because of the domain, not because of 18.

- **`timestamptz` plus an event time zone.** Instants are absolute. The zone is how wall-clock time is recovered. [ADR 0005](adr/0005-timestamptz-and-event-zone.md).
- **GiST exclusion on room occupancy** (`btree_gist`, `'[)'`). Two sessions cannot occupy the same room at the same time. [ADR 0002](adr/0002-room-conflicts-and-version.md).
