import { useSuspenseQuery } from "@tanstack/react-query"
import { createFileRoute } from "@tanstack/react-router"

import { Button } from "@/components/ui/button"
import { healthQuery } from "@/lib/query/health"

export const Route = createFileRoute("/")({
  loader: ({ context }) => context.queryClient.ensureQueryData(healthQuery),
  component: Home,
  errorComponent: HealthError,
})

function HealthError({ error }: { error: Error }) {
  return (
    <main className="mx-auto max-w-xl p-8">
      <h1 className="text-2xl font-semibold">Event management</h1>
      <p className="mt-2 text-neutral-600">
        Typed GET /healthz did not succeed. Start the API with `make up`.
      </p>
      <p className="mt-6 font-mono text-sm">{error.message}</p>
    </main>
  )
}

function Home() {
  const { data } = useSuspenseQuery(healthQuery)

  return (
    <main className="mx-auto max-w-xl p-8">
      <h1 className="text-2xl font-semibold">Event management</h1>
      <p className="mt-2 text-neutral-600">
        Frontend scaffold. Schedule, invitations, and the agent UI are not built
        yet.
      </p>
      <p className="mt-6 font-mono text-sm">
        GET /healthz → {data.status}
      </p>
      <Button className="mt-6" type="button" disabled>
        Placeholder
      </Button>
    </main>
  )
}
