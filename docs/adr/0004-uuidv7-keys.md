# ADR 0004 — uuidv7 Primary Keys

## Decision

Domain table IDs are `uuid DEFAULT uuidv7()`.

uuidv7 values are time-ordered, so `id` already sorts by creation time. Invitations therefore have one index that is both chronological order and a stable keyset:

```sql
CREATE INDEX invitations_event_id_id_idx ON invitations (event_id, id);
```

Random UUIDs would not give that order from the primary key.

## Rejected Alternatives

**Random UUIDs (`uuidv4`)**

Fine as identifiers. Useless as an order. A list that should be chronological would then need a separate `created_at` column and a wider index just to sort.

**A `created_at` column used only for ordering**

Would work, but it duplicates order already present in a uuidv7 primary key.

## Why

Invitations are expected to be listed in the order they were created. Putting that order in the ID keeps one index, not two.

The seeder generates uuidv7 values in Go rather than using `DEFAULT uuidv7()`. COPY needs parent IDs before child rows (rooms and sessions point at events), and a database-assigned default would make seed reruns non-reproducible.

## Evidence

Captured 2026-08-17 against the 50k-invitation seed (1,000 rows on one event). First page and keyset page both use `invitations_event_id_id_idx` and stop after 50 rows — no sequential scan of the table.

First page:

```sql
SELECT id, event_id, email, status, created_at
FROM invitations
WHERE event_id = $1
ORDER BY event_id, id
LIMIT 50;
```

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

Keyset page:

```sql
SELECT id, event_id, email, status, created_at
FROM invitations
WHERE event_id = $1 AND id > $2
ORDER BY event_id, id
LIMIT 50;
```

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

Same plan is in [docs/postgres-18.md](../postgres-18.md).
