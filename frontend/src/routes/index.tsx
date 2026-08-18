import { useSuspenseQuery } from "@tanstack/react-query"
import { Link, createFileRoute } from "@tanstack/react-router"

import { ApiErrorView } from "@/components/api/ApiErrorView"
import { SignInForm } from "@/components/auth/SignInForm"
import { eventsQuery } from "@/lib/query/events"
import { healthQuery } from "@/lib/query/health"
import { instantToYmd } from "@/lib/tz/eventZone"

export const Route = createFileRoute("/")({
  loader: ({ context }) =>
    Promise.all([
      context.queryClient.ensureQueryData(healthQuery),
      context.queryClient.ensureQueryData(eventsQuery),
    ]),
  component: Home,
  pendingComponent: HomePending,
  errorComponent: ApiErrorView,
})

function HomePending() {
  return (
    <main className="mx-auto max-w-xl p-8">
      <p className="text-neutral-600">Loading events…</p>
    </main>
  )
}

function Home() {
  const { data: health } = useSuspenseQuery(healthQuery)
  const { data: page } = useSuspenseQuery(eventsQuery)

  return (
    <main className="mx-auto max-w-xl p-8">
      <h1 className="text-2xl font-semibold">Event management</h1>
      <p className="mt-2 text-neutral-600">
        Pick an event, or open the{" "}
        <Link to="/agent" className="underline">
          agent
        </Link>
        .
      </p>
      <p className="mt-4 font-mono text-sm">GET /healthz → {health.status}</p>

      {page.items.length === 0 ? (
        <p className="mt-6 text-neutral-700">No events you can read.</p>
      ) : (
        <ul className="mt-6 divide-y rounded-md border">
          {page.items.map((event) => (
            <li key={event.id}>
              <Link
                to="/events/$eventId/schedule"
                params={{ eventId: event.id }}
                className="block px-3 py-2 hover:bg-neutral-50"
              >
                <span className="font-medium">{event.name}</span>
                <span className="mt-0.5 block font-mono text-xs text-neutral-500">
                  {event.timeZone} · {instantToYmd(event.startsAt, event.timeZone)}
                </span>
              </Link>
            </li>
          ))}
        </ul>
      )}

      <details className="mt-8">
        <summary className="cursor-pointer text-sm text-neutral-600">
          Sign in as someone else
        </summary>
        <div className="mt-3">
          <SignInForm />
        </div>
      </details>
    </main>
  )
}
