import type { ReactNode } from "react"
import {
  DndContext,
  PointerSensor,
  closestCenter,
  useDroppable,
  useSensor,
  useSensors,
  type DragEndEvent,
} from "@dnd-kit/core"

import type { Event, Room, Session } from "@/lib/api/types"
import { SessionBlock } from "@/components/schedule/SessionBlock"
import { HOUR_HEIGHT, hourRange, placeSessions } from "@/lib/schedule/layout"
import { dropToMove, type MoveSessionInput } from "@/lib/schedule/drop"
import { sameEventDay } from "@/lib/tz/eventZone"

type ScheduleGridProps = {
  event: Event
  rooms: Room[]
  sessions: Session[]
  day: string
  onReschedule?: (input: MoveSessionInput) => void
}

function RoomColumn({
  id,
  height,
  children,
}: {
  id: string
  height: number
  children: ReactNode
}) {
  const { setNodeRef, isOver } = useDroppable({ id })
  return (
    <div
      ref={setNodeRef}
      data-testid={`room-column-${id}`}
      className={
        isOver
          ? "relative border-l bg-amber-50 bg-[linear-gradient(to_bottom,transparent_calc(100%-1px),rgb(229_229_229)_1px)]"
          : "relative border-l bg-[linear-gradient(to_bottom,transparent_calc(100%-1px),rgb(229_229_229)_1px)]"
      }
      style={{
        height,
        backgroundSize: `100% ${HOUR_HEIGHT}px`,
      }}
    >
      {children}
    </div>
  )
}

export function ScheduleGrid({
  event,
  rooms,
  sessions,
  day,
  onReschedule,
}: ScheduleGridProps) {
  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 6 } }),
  )
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

  function handleDragEnd(eventDnD: DragEndEvent) {
    if (!onReschedule || !eventDnD.over) return
    const session = eventDnD.active.data.current?.session as Session | undefined
    if (!session) return
    const overId = String(eventDnD.over.id)
    const roomId = overId === "unplaced" ? null : overId
    const input = dropToMove({
      session,
      timeZone: event.timeZone,
      day,
      roomId,
      deltaY: eventDnD.delta.y,
      hourHeight: HOUR_HEIGHT,
    })
    if (input) onReschedule(input)
  }

  return (
    <DndContext
      sensors={sensors}
      collisionDetection={closestCenter}
      onDragEnd={handleDragEnd}
    >
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
            <RoomColumn key={col.id} id={col.id} height={height}>
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
                  draggable={Boolean(onReschedule)}
                />
              ))}
            </RoomColumn>
          ))}
        </div>
      </div>
    </DndContext>
  )
}
