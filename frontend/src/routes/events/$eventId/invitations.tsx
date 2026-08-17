import { useSuspenseQuery } from "@tanstack/react-query"
import { Link, createFileRoute } from "@tanstack/react-router"

import { ApiErrorView } from "@/components/api/ApiErrorView"
import { InvitationList } from "@/components/invitations/InvitationList"
import { eventQuery } from "@/lib/query/events"
import { invitationsQuery } from "@/lib/query/invitations"

export const Route = createFileRoute("/events/$eventId/invitations")({
  loader: ({ context, params }) =>
    Promise.all([
      context.queryClient.ensureQueryData(eventQuery(params.eventId)),
      context.queryClient.ensureInfiniteQueryData(invitationsQuery(params.eventId)),
    ]),
  pendingComponent: InvitationsPending,
  errorComponent: ApiErrorView,
  component: InvitationsPage,
})

function InvitationsPending() {
  return (
    <main className="mx-auto max-w-4xl p-8">
      <p className="text-neutral-600">Loading invitations…</p>
    </main>
  )
}

function InvitationsPage() {
  const { eventId } = Route.useParams()
  const { data: event } = useSuspenseQuery(eventQuery(eventId))

  return (
    <main className="mx-auto max-w-4xl p-6">
      <p className="text-sm text-neutral-500">
        <Link to="/" className="underline">
          Events
        </Link>
        {" · "}
        <Link
          to="/events/$eventId/schedule"
          params={{ eventId }}
          className="underline"
        >
          Schedule
        </Link>
      </p>
      <h1 className="mt-2 text-2xl font-semibold">{event.name}</h1>
      <p className="mt-1 text-sm text-neutral-600">
        Keyset pages of 50. Only the visible rows are in the DOM. Position in
        the URL is not wired yet.
      </p>
      <div className="mt-6">
        <InvitationList eventId={eventId} />
      </div>
    </main>
  )
}
