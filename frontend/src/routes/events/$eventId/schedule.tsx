import { useSuspenseQuery } from "@tanstack/react-query"
import { Link, createFileRoute } from "@tanstack/react-router"

import { ApiErrorView } from "@/components/api/ApiErrorView"
import { DayNav } from "@/components/schedule/DayNav"
import { ScheduleBoard } from "@/components/schedule/ScheduleBoard"
import {
  NavSep,
  PageFrame,
  PageLead,
  PageNav,
  PageTitle,
  navLinkClass,
} from "@/components/layout/PageFrame"
import { Skeleton } from "@/components/ui/skeleton"
import { eventQuery } from "@/lib/query/events"
import { roomsQuery } from "@/lib/query/rooms"
import { sessionsQuery } from "@/lib/query/sessions"
import { instantToYmd } from "@/lib/tz/eventZone"

const ymd = /^\d{4}-\d{2}-\d{2}$/

export const Route = createFileRoute("/events/$eventId/schedule")({
  validateSearch: (search: Record<string, unknown>): { day?: string } => {
    const day = typeof search.day === "string" && ymd.test(search.day) ? search.day : undefined
    return { day }
  },
  loader: ({ context, params }) =>
    Promise.all([
      context.queryClient.ensureQueryData(eventQuery(params.eventId)),
      context.queryClient.ensureQueryData(roomsQuery(params.eventId)),
      context.queryClient.ensureQueryData(sessionsQuery(params.eventId)),
    ]),
  pendingComponent: SchedulePending,
  errorComponent: ApiErrorView,
  component: SchedulePage,
})

function SchedulePending() {
  return (
    <PageFrame width="6xl">
      <p className="text-neutral-600">Loading schedule…</p>
      <Skeleton className="mt-4 h-64 w-full" />
    </PageFrame>
  )
}

function SchedulePage() {
  const { eventId } = Route.useParams()
  const { day: daySearch } = Route.useSearch()
  const { data: event } = useSuspenseQuery(eventQuery(eventId))
  const { data: roomPage } = useSuspenseQuery(roomsQuery(eventId))
  const { data: sessionPage } = useSuspenseQuery(sessionsQuery(eventId))

  const sessions = sessionPage.items
  const rooms = roomPage.items
  const day =
    daySearch ??
    (sessions[0]
      ? instantToYmd(sessions[0].startsAt, event.timeZone)
      : instantToYmd(event.startsAt, event.timeZone))
  const onDay = sessions.filter(
    (session) => instantToYmd(session.startsAt, event.timeZone) === day,
  )

  return (
    <PageFrame width="6xl">
      <PageNav>
        <Link to="/" className={navLinkClass}>
          Events
        </Link>
        <NavSep />
        <Link
          to="/events/$eventId/invitations"
          params={{ eventId }}
          className={navLinkClass}
        >
          Invitations
        </Link>
        <NavSep />
        <Link to="/agent" className={navLinkClass}>
          Agent
        </Link>
      </PageNav>
      <PageTitle>{event.name}</PageTitle>
      <PageLead>
        Times in <span className="font-mono">{event.timeZone}</span>, not the
        browser zone. Drag a session to a new room or time. Rejections roll back
        to server truth.
      </PageLead>

      {sessions.length === 0 ? (
        <p className="mt-8 rounded-xl border border-dashed border-neutral-200 bg-white px-4 py-8 text-center text-sm text-neutral-700">
          No sessions on this event.
        </p>
      ) : (
        <>
          <div className="mt-6">
            <DayNav eventId={eventId} day={day} />
          </div>
          {onDay.length === 0 ? (
            <p className="mt-8 rounded-xl border border-dashed border-neutral-200 bg-white px-4 py-8 text-center text-sm text-neutral-700">
              No sessions on {day}.
            </p>
          ) : (
            <div className="mt-4">
              <ScheduleBoard event={event} rooms={rooms} day={day} />
            </div>
          )}
        </>
      )}
    </PageFrame>
  )
}
