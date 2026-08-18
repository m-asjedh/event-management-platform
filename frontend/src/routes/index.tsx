import { useSuspenseQuery } from "@tanstack/react-query"
import { Link, createFileRoute } from "@tanstack/react-router"

import { ApiErrorView } from "@/components/api/ApiErrorView"
import { SignInForm } from "@/components/auth/SignInForm"
import {
  PageFrame,
  PageLead,
  PageTitle,
  navLinkClass,
} from "@/components/layout/PageFrame"
import { Card, CardContent } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
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
    <PageFrame>
      <p className="text-neutral-600">Loading events…</p>
      <div className="mt-4 space-y-2">
        <Skeleton className="h-16 w-full" />
        <Skeleton className="h-16 w-full" />
        <Skeleton className="h-16 w-full" />
      </div>
    </PageFrame>
  )
}

function Home() {
  const { data: health } = useSuspenseQuery(healthQuery)
  const { data: page } = useSuspenseQuery(eventsQuery)

  return (
    <PageFrame>
      <PageTitle>Event management</PageTitle>
      <PageLead>
        Pick an event, or open the{" "}
        <Link to="/agent" className={navLinkClass}>
          agent
        </Link>
        .
      </PageLead>
      <p className="mt-4 font-mono text-xs text-neutral-500">
        GET /healthz → {health.status}
      </p>

      {page.items.length === 0 ? (
        <p className="mt-6 rounded-xl border border-dashed border-neutral-200 bg-white px-4 py-8 text-center text-sm text-neutral-700">
          No events you can read.
        </p>
      ) : (
        <ul className="mt-6 grid gap-3">
          {page.items.map((event) => (
            <li key={event.id}>
              <Link
                to="/events/$eventId/schedule"
                params={{ eventId: event.id }}
                className="block rounded-xl focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-neutral-900"
              >
                <Card className="transition-colors hover:border-neutral-300 hover:bg-neutral-50">
                  <CardContent className="flex items-center justify-between gap-4 py-4">
                    <span>
                      <span className="block font-medium text-neutral-900">
                        {event.name}
                      </span>
                      <span className="mt-0.5 block font-mono text-xs text-neutral-500">
                        {event.timeZone} · {instantToYmd(event.startsAt, event.timeZone)}
                      </span>
                    </span>
                    <span className="text-xs font-medium text-neutral-400">Open →</span>
                  </CardContent>
                </Card>
              </Link>
            </li>
          ))}
        </ul>
      )}

      <details className="mt-8">
        <summary className="cursor-pointer text-sm text-neutral-600 hover:text-neutral-900">
          Sign in as someone else
        </summary>
        <div className="mt-3">
          <SignInForm />
        </div>
      </details>
    </PageFrame>
  )
}
