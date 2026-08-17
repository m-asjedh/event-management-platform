import type { Event, Room, Session } from "@/lib/api/types"
import { SessionBlock } from "@/components/schedule/SessionBlock"
import { HOUR_HEIGHT, hourRange, placeSessions } from "@/lib/schedule/layout"
import { sameEventDay } from "@/lib/tz/eventZone"

type ScheduleGridProps = {
  event: Event
  rooms: Room[]
  sessions: Session[]
  day: string
}

export function ScheduleGrid({ event, rooms, sessions, day }: ScheduleGridProps) {
  const onDay = sessions.filter((session) =>
    sameEventDay(session.startsAt, event.timeZone, day),
  )
  const placed = placeSessions(onDay, event.timeZone)
  const { startHour, endHour } = hourRange(placed)
  const hours = Array.from({ length: endHour - startHour }, (_, i) => startHour + i)
  const gridStartMin = startHour * 60
  const height = hours.length * HOUR_HEIGHT

  const roomName = (session: Session): string => {
    if (!session.roomId) return "Unplaced"
    return rooms.find((room) => room.id === session.roomId)?.name ?? "Unplaced"
  }

  const columns = [
    ...rooms.map((room) => ({
      id: room.id,
      name: room.name,
      items: placed.filter((item) => item.session.roomId === room.id),
    })),
    {
      id: "unplaced",
      name: "Unplaced",
      items: placed.filter((item) => !item.session.roomId),
    },
  ]

  return (
    <div className="overflow-x-auto">
      <div
        className="grid min-w-160"
        style={{
          gridTemplateColumns: `3.5rem repeat(${columns.length}, minmax(8rem, 1fr))`,
        }}
      >
        <div className="sticky top-0 z-10 border-b bg-neutral-50" />
        {columns.map((col) => (
          <div
            key={col.id}
            className="sticky top-0 z-10 border-b border-l bg-neutral-50 px-2 py-2 text-sm font-medium"
          >
            {col.name}
          </div>
        ))}

        <div className="relative border-r" style={{ height }}>
          {hours.map((hour) => (
            <div
              key={hour}
              className="absolute right-1 text-[11px] tabular-nums text-neutral-500"
              style={{ top: (hour - startHour) * HOUR_HEIGHT - 7 }}
            >
              {`${hour.toString().padStart(2, "0")}:00`}
            </div>
          ))}
        </div>

        {columns.map((col) => (
          <div
            key={col.id}
            data-testid={`room-column-${col.id}`}
            className="relative border-l bg-[linear-gradient(to_bottom,transparent_calc(100%-1px),rgb(229_229_229)_1px)]"
            style={{
              height,
              backgroundSize: `100% ${HOUR_HEIGHT}px`,
            }}
          >
            {col.items.map((item) => (
              <SessionBlock
                key={item.session.id}
                session={item.session}
                roomName={roomName(item.session)}
                timeZone={event.timeZone}
                conflict={item.conflict}
                lane={item.lane}
                lanes={item.lanes}
                top={((item.startMin - gridStartMin) / 60) * HOUR_HEIGHT}
                height={Math.max(
                  ((item.endMin - item.startMin) / 60) * HOUR_HEIGHT,
                  36,
                )}
              />
            ))}
          </div>
        ))}
      </div>
    </div>
  )
}
