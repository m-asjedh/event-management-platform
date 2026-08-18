Vite + TypeScript + TanStack Router + TanStack Query + Tailwind + shadcn/ui.

Types come from `openapi/openapi.yaml`. Do not edit `src/generated/api.ts`.

`make up && make seed && make test` is how reviewers verify the build. It does not serve the UI.

To see the UI:

```bash
make up && make seed && make frontend
```

Then open http://localhost:5173, sign in, and open an event's week schedule at
`/events/$eventId/schedule`. Times are the event IANA zone. Drag to reschedule;
the server is the authority after a rejection. Invitations: `/events/$eventId/invitations`.
A shared `?status=&cursor=` link is the same filter plus the same keyset slice; cursor is
the opaque token, not a scroll pixel. Agent: `/agent` — writes wait on the exact JSON.

Regenerate types after a spec change:

```bash
make gen-api
```

```bash
make frontend-test     # Vitest
make frontend-build    # tsc + vite build
make check-generated   # fail if generated types are stale
```

CI runs `make check-generated` and `make frontend-build`.
The frontend is not part of `make up`, so the under-5-minute backend path stays the same.
