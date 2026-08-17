import { useDraggable } from "@dnd-kit/core"
import { CSS } from "@dnd-kit/utilities"

import type { Session } from "@/lib/api/types"
import { cn } from "@/lib/utils"
import { formatEventTime } from "@/lib/tz/eventZone"

type SessionBlockProps = {
  session: Session
  roomName: string
  timeZone: string
  conflict: boolean
  lane: number
  lanes: number
  top: number
  height: number
  draggable: boolean
}

export function SessionBlock({
  session,
  roomName,
  timeZone,
  conflict,
  lane,
  lanes,
  top,
  height,
  draggable,
}: SessionBlockProps) {
  const start = formatEventTime(session.startsAt, timeZone)
  const end = formatEventTime(session.endsAt, timeZone)
  const label = `${session.title}, ${roomName}, ${start} to ${end}${conflict ? ", room conflict" : ""}`
  const { attributes, listeners, setNodeRef, transform, isDragging } = useDraggable({
    id: session.id,
    data: { session },
    disabled: !draggable,
  })

  return (
    <article
      ref={setNodeRef}
      aria-label={label}
      data-session-id={session.id}
      data-conflict={conflict ? "true" : "false"}
      data-start-local={start}
      data-end-local={end}
      data-room-id={session.roomId ?? "unplaced"}
      data-version={String(session.version)}
      className={cn(
        "absolute overflow-hidden rounded-md border px-1.5 py-1 text-left shadow-sm",
        isDragging && "z-20 opacity-70",
        conflict
          ? "border-red-700 bg-[image:repeating-linear-gradient(-45deg,rgba(185,28,28,0.16)_0_8px,rgba(254,243,199,0.95)_8px_16px)]"
          : "border-neutral-300 bg-white",
      )}
      style={{
        top,
        height,
        left: `calc(${(lane / lanes) * 100}% + 2px)`,
        width: `calc(${(1 / lanes) * 100}% - 4px)`,
        transform: CSS.Translate.toString(transform),
      }}
    >
      {draggable ? (
        <button
          type="button"
          className="absolute inset-0 cursor-grab active:cursor-grabbing"
          aria-label={`Drag ${session.title}`}
          {...listeners}
          {...attributes}
        />
      ) : null}
      <div className="relative z-10 pointer-events-none">
        <h3 className="truncate text-xs font-semibold text-neutral-900">
          {session.title}
        </h3>
        <p className="truncate text-[11px] text-neutral-600">{roomName}</p>
        <p className="text-[11px] tabular-nums text-neutral-700">
          {start}–{end}
        </p>
        {conflict ? (
          <p className="mt-0.5 text-[11px] font-semibold text-red-800">Conflict</p>
        ) : null}
      </div>
    </article>
  )
}
