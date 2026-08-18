import { useSuspenseQuery } from "@tanstack/react-query"
import { Link, createFileRoute } from "@tanstack/react-router"
import { useCallback } from "react"

import { ApiErrorView } from "@/components/api/ApiErrorView"
import { InvitationList } from "@/components/invitations/InvitationList"
import {
  NavSep,
  PageFrame,
  PageLead,
  PageNav,
  PageTitle,
  navLinkClass,
} from "@/components/layout/PageFrame"
import { Skeleton } from "@/components/ui/skeleton"
import type { InvitationCursor, InvitationStatus } from "@/lib/api/types"
import {
  INVITATION_STATUSES,
  isInvitationStatus,
  validateInvitationsSearch,
} from "@/lib/invitations/search"
import { eventQuery } from "@/lib/query/events"
import { invitationsQuery } from "@/lib/query/invitations"

export const Route = createFileRoute("/events/$eventId/invitations")({
  validateSearch: validateInvitationsSearch,
  loaderDeps: ({ search }) => ({
    status: search.status,
    cursor: search.cursor,
  }),
  loader: ({ context, params, deps }) =>
    Promise.all([
      context.queryClient.ensureQueryData(eventQuery(params.eventId)),
      context.queryClient.ensureInfiniteQueryData(
        invitationsQuery(params.eventId, {
          cursor: deps.cursor,
          status: deps.status,
        }),
      ),
    ]),
  pendingComponent: InvitationsPending,
  errorComponent: ApiErrorView,
  component: InvitationsPage,
})

function InvitationsPending() {
  return (
    <PageFrame width="4xl">
      <p className="text-neutral-600">Loading invitations…</p>
      <Skeleton className="mt-4 h-96 w-full" />
    </PageFrame>
  )
}

function InvitationsPage() {
  const { eventId } = Route.useParams()
  const { status, cursor } = Route.useSearch()
  const navigate = Route.useNavigate()
  const { data: event } = useSuspenseQuery(eventQuery(eventId))

  const onStartCursorChange = useCallback(
    (next: InvitationCursor) => {
      if (next === cursor) return
      void navigate({
        replace: true,
        search: (prev: { status?: InvitationStatus; cursor?: InvitationCursor }) => ({
          ...prev,
          cursor: next,
        }),
      })
    },
    [cursor, navigate],
  )

  const onStatusChange = (next: InvitationStatus | undefined) => {
    void navigate({
      replace: true,
      search: { status: next, cursor: undefined },
    })
  }

  return (
    <PageFrame width="4xl">
      <PageNav>
        <Link to="/" className={navLinkClass}>
          Events
        </Link>
        <NavSep />
        <Link
          to="/events/$eventId/schedule"
          params={{ eventId }}
          className={navLinkClass}
        >
          Schedule
        </Link>
        <NavSep />
        <Link to="/agent" className={navLinkClass}>
          Agent
        </Link>
      </PageNav>
      <PageTitle>{event.name}</PageTitle>
      <PageLead>
        Position is the opaque keyset cursor in the URL, not a pixel or page
        number. Status filters the pages already loaded — the list API has no
        status parameter.
      </PageLead>
      <div className="mt-4">
        <label className="flex items-center gap-2 text-sm">
          <span className="text-neutral-600">Status</span>
          <select
            aria-label="Invitation status"
            className="h-9 rounded-md border border-neutral-200 bg-white px-3 text-sm shadow-sm outline-none focus-visible:ring-2 focus-visible:ring-neutral-900/10"
            value={status ?? "all"}
            onChange={(event) => {
              const value = event.target.value
              onStatusChange(isInvitationStatus(value) ? value : undefined)
            }}
          >
            <option value="all">All</option>
            {INVITATION_STATUSES.map((value) => (
              <option key={value} value={value}>
                {value}
              </option>
            ))}
          </select>
        </label>
      </div>
      <div className="mt-6">
        <InvitationList
          eventId={eventId}
          cursor={cursor}
          status={status}
          onStartCursorChange={onStartCursorChange}
        />
      </div>
    </PageFrame>
  )
}
