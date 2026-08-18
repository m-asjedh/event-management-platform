import { useDraggable } from "@dnd-kit/core"
import { CSS } from "@dnd-kit/utilities"
import { GripVertical } from "lucide-react"

import { Badge } from "@/components/ui/badge"
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
  moving?: boolean
  reverting?: boolean
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
  moving = false,
  reverting = false,
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
        "absolute overflow-hidden rounded-md border px-1.5 py-1 text-left shadow-sm transition-[opacity,box-shadow]",
        isDragging && "z-20 cursor-grabbing opacity-80 ring-2 ring-neutral-900/20",
        moving && "z-10 opacity-70 ring-2 ring-dashed ring-neutral-400",
        reverting && "snap-back-ring",
        conflict
          ? "border-red-600 bg-red-50"
          : "border-neutral-200 bg-white",
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
      <div className="relative z-10 pointer-events-none flex gap-1">
        {draggable ? (
          <GripVertical className="mt-0.5 size-3 shrink-0 text-neutral-400" aria-hidden />
        ) : null}
        <div className="min-w-0 flex-1">
          <h3 className="truncate text-xs font-semibold text-neutral-900">
            {session.title}
          </h3>
          <p className="truncate text-[11px] text-neutral-600">{roomName}</p>
          <p className="text-[11px] tabular-nums text-neutral-700">
            {start}–{end}
          </p>
          {conflict ? (
            <Badge variant="destructive" className="mt-0.5 normal-case tracking-normal">
              Conflict
            </Badge>
          ) : null}
        </div>
      </div>
    </article>
  )
}
