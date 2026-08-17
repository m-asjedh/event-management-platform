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
