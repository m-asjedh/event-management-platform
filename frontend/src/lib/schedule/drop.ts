import type { Session } from "@/lib/api/types"
import { minutesFromMidnight, minutesToWall } from "@/lib/tz/eventZone"

export const SNAP_MINUTES = 15

export type MoveSessionInput = {
  session: Session
  roomId: string | null
  startLocal: string
  endLocal: string
}

export function snapMinutes(value: number, step = SNAP_MINUTES): number {
  return Math.round(value / step) * step
}

export function dropToMove(opts: {
  session: Session
  timeZone: string
  day: string
  roomId: string | null
  deltaY: number
  hourHeight: number
}): MoveSessionInput | null {
  const startMin = minutesFromMidnight(opts.session.startsAt, opts.timeZone)
  const endMin = minutesFromMidnight(opts.session.endsAt, opts.timeZone)
  const duration = Math.max(endMin - startMin, SNAP_MINUTES)
  const nextStart = snapMinutes(startMin + (opts.deltaY / opts.hourHeight) * 60)
  const clamped = Math.max(0, Math.min(nextStart, 24 * 60 - duration))
  const startLocal = minutesToWall(opts.day, clamped)
  const endLocal = minutesToWall(opts.day, clamped + duration)
  const currentRoom = opts.session.roomId ?? null
  const currentStart = minutesToWall(opts.day, startMin)
  if (currentRoom === opts.roomId && currentStart === startLocal) {
    return null
  }
  return {
    session: opts.session,
    roomId: opts.roomId,
    startLocal,
    endLocal,
  }
}
