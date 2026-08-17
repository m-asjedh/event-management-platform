import type { Session } from "@/lib/api/types"

/** Half-open [start, end), matching the room exclusion tstzrange. */
export function intervalsOverlap(
  aStart: string,
  aEnd: string,
  bStart: string,
  bEnd: string,
): boolean {
  const a0 = Date.parse(aStart)
  const a1 = Date.parse(aEnd)
  const b0 = Date.parse(bStart)
  const b1 = Date.parse(bEnd)
  return a0 < b1 && b0 < a1
}

/**
 * Same-room overlaps. Unplaced sessions (no roomId) are not a room clash —
 * speaker clashes are not in GET /sessions, so they cannot be marked here.
 */
export function conflictingSessionIds(sessions: Session[]): Set<string> {
  const ids = new Set<string>()
  const byRoom = new Map<string, Session[]>()
  for (const session of sessions) {
    if (!session.roomId) continue
    const list = byRoom.get(session.roomId) ?? []
    list.push(session)
    byRoom.set(session.roomId, list)
  }
  for (const group of byRoom.values()) {
    for (let i = 0; i < group.length; i++) {
      for (let j = i + 1; j < group.length; j++) {
        const a = group[i]
        const b = group[j]
        if (intervalsOverlap(a.startsAt, a.endsAt, b.startsAt, b.endsAt)) {
          ids.add(a.id)
          ids.add(b.id)
        }
      }
    }
  }
  return ids
}
