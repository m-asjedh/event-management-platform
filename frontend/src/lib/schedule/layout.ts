import type { Session } from "@/lib/api/types"
import { conflictingSessionIds } from "@/lib/schedule/overlaps"
import { minutesFromMidnight } from "@/lib/tz/eventZone"

export const HOUR_HEIGHT = 72

export type PlacedSession = {
  session: Session
  conflict: boolean
  lane: number
  lanes: number
  startMin: number
  endMin: number
}

function packLanes(group: Session[]): Map<string, { lane: number; lanes: number }> {
  const sorted = [...group].sort(
    (a, b) => Date.parse(a.startsAt) - Date.parse(b.startsAt) || a.id.localeCompare(b.id),
  )
  const laneEnds: number[] = []
  const laneOf = new Map<string, number>()
  for (const session of sorted) {
    const start = Date.parse(session.startsAt)
    const end = Date.parse(session.endsAt)
    let lane = laneEnds.findIndex((until) => until <= start)
    if (lane === -1) {
      lane = laneEnds.length
      laneEnds.push(end)
    } else {
      laneEnds[lane] = end
    }
    laneOf.set(session.id, lane)
  }
  const lanes = Math.max(1, laneEnds.length)
  const out = new Map<string, { lane: number; lanes: number }>()
  for (const session of sorted) {
    out.set(session.id, { lane: laneOf.get(session.id) ?? 0, lanes })
  }
  return out
}

export function placeSessions(
  sessions: Session[],
  timeZone: string,
): PlacedSession[] {
  const conflicts = conflictingSessionIds(sessions)
  const byColumn = new Map<string, Session[]>()
  for (const session of sessions) {
    const key = session.roomId ?? "unplaced"
    const list = byColumn.get(key) ?? []
    list.push(session)
    byColumn.set(key, list)
  }
  const placed: PlacedSession[] = []
  for (const group of byColumn.values()) {
    const lanes = packLanes(group)
    for (const session of group) {
      const packing = lanes.get(session.id) ?? { lane: 0, lanes: 1 }
      placed.push({
        session,
        conflict: conflicts.has(session.id),
        lane: packing.lane,
        lanes: packing.lanes,
        startMin: minutesFromMidnight(session.startsAt, timeZone),
        endMin: minutesFromMidnight(session.endsAt, timeZone),
      })
    }
  }
  return placed
}

export function hourRange(
  placed: PlacedSession[],
): { startHour: number; endHour: number } {
  let startHour = 9
  let endHour = 17
  for (const item of placed) {
    startHour = Math.min(startHour, Math.floor(item.startMin / 60))
    endHour = Math.max(endHour, Math.ceil(item.endMin / 60))
  }
  if (endHour <= startHour) {
    endHour = startHour + 1
  }
  return { startHour, endHour }
}
