# Frontend

Vite + TypeScript + TanStack Router + TanStack Query + Tailwind + shadcn/ui.

Types come from `openapi/openapi.yaml`. Do not edit `src/generated/api.ts`.

`make up && make seed && make test` is how reviewers verify the build. It does not serve the UI.

To see the UI:

```bash
make up && make seed && make frontend
```

Then open http://localhost:5173.

Regenerate types after a spec change:

```bash
make gen-api
```

CI runs `make check-generated` (fails if that file is stale) and `make frontend-build`.
The frontend is not part of `make up`, so the under-5-minute backend path stays the same.
