import { describe, expect, it } from "vitest"

import type { Session } from "@/lib/api/types"
import { conflictingSessionIds } from "@/lib/schedule/overlaps"

function session(partial: Pick<Session, "id" | "roomId" | "startsAt" | "endsAt">): Session {
  return {
    eventId: "evt_1",
    title: partial.id,
    description: "",
    version: 1,
    ...partial,
  }
}

describe("conflictingSessionIds", () => {
  it("flags an overlapping same-room pair and ignores a touching neighbour", () => {
    const overlapA = session({
      id: "s1",
      roomId: "room-a",
      startsAt: "2026-03-08T13:00:00.000Z",
      endsAt: "2026-03-08T14:00:00.000Z",
    })
    const overlapB = session({
      id: "s2",
      roomId: "room-a",
      startsAt: "2026-03-08T13:30:00.000Z",
      endsAt: "2026-03-08T14:30:00.000Z",
    })
    const adjacent = session({
      id: "s3",
      roomId: "room-a",
      startsAt: "2026-03-08T14:30:00.000Z",
      endsAt: "2026-03-08T15:30:00.000Z",
    })
    const otherRoom = session({
      id: "s4",
      roomId: "room-b",
      startsAt: "2026-03-08T13:00:00.000Z",
      endsAt: "2026-03-08T14:00:00.000Z",
    })
    const unplaced = session({
      id: "s5",
      roomId: null,
      startsAt: "2026-03-08T13:00:00.000Z",
      endsAt: "2026-03-08T14:00:00.000Z",
    })

    const ids = conflictingSessionIds([
      overlapA,
      overlapB,
      adjacent,
      otherRoom,
      unplaced,
    ])
    expect(ids).toEqual(new Set(["s1", "s2"]))

    const touching = conflictingSessionIds([
      session({
        id: "t1",
        roomId: "room-a",
        startsAt: "2026-03-08T13:00:00.000Z",
        endsAt: "2026-03-08T14:00:00.000Z",
      }),
      session({
        id: "t2",
        roomId: "room-a",
        startsAt: "2026-03-08T14:00:00.000Z",
        endsAt: "2026-03-08T15:00:00.000Z",
      }),
    ])
    expect(touching.size).toBe(0)
  })
})
