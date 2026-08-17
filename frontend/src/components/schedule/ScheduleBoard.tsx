import { useSuspenseQuery } from "@tanstack/react-query"

import { MoveNoticeBanner } from "@/components/schedule/MoveNoticeBanner"
import { ScheduleGrid } from "@/components/schedule/ScheduleGrid"
import type { Event, Room } from "@/lib/api/types"
import { useMoveSession } from "@/lib/query/moveSession"
import { sessionsQuery } from "@/lib/query/sessions"

export function ScheduleBoard({
  event,
  rooms,
  day,
}: {
  event: Event
  rooms: Room[]
  day: string
}) {
  const { data } = useSuspenseQuery(sessionsQuery(event.id))
  const move = useMoveSession({
    eventId: event.id,
    timeZone: event.timeZone,
    rooms,
  })

  return (
    <div>
      <MoveNoticeBanner notice={move.notice} onDismiss={move.clearNotice} />
      <ScheduleGrid
        event={event}
        rooms={rooms}
        sessions={data.items}
        day={day}
        onReschedule={move.mutate}
      />
    </div>
  )
}
