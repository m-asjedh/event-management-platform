# ADR 0006 — Frontend Data Loading and Cache Strategy

## Decision

TanStack Router owns **where the user is**: navigation, path params, and typed search
state (`?day=` on the schedule). The route loader preloads data the page cannot render
without, via `ensureQueryData`.

TanStack Query owns **what server data they are looking at**: caches, refetch, and
mutations with optimistic update and rollback. Drag-to-reschedule patches
`/sessions/{id}` and reconciles from the typed error envelope (`ROOM_CONFLICT`
rolls back the snapshot; `STALE_VERSION` applies `conflict.currentState`).

The schedule page uses the same query options in the loader and in
`useSuspenseQuery`, so the component stays subscribed to the cache.

## Rejected Alternatives

**Fetch only inside components.** Shared URL state and route-level loading become ad hoc.

**Keep all server data only in route loaders.** Fine for first paint. A poor place for
drag-to-reschedule: no mutation lifecycle, no optimistic cache, no rollback that leaves
the rest of the page in place.

## Why

```text
TanStack Router  → where the user is
TanStack Query   → what server data the user is looking at
```

That split is what lets a later `ROOM_CONFLICT` / `STALE_VERSION` recovery (ADR 0002)
target the cache without remounting the week view.
