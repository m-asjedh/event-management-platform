import { describe, expect, it } from "vitest"

import type { Session } from "@/lib/api/types"
import { dropToMove } from "@/lib/schedule/drop"
import { HOUR_HEIGHT } from "@/lib/schedule/layout"

const session: Session = {
  id: "s1",
  eventId: "e1",
  title: "Talk",
  description: "",
  roomId: "room-a",
  startsAt: "2026-03-08T13:00:00.000Z",
  endsAt: "2026-03-08T14:00:00.000Z",
  version: 1,
}

describe("dropToMove", () => {
  it("keeps duration and emits event-local wall clock for a one-hour drop", () => {
    const input = dropToMove({
      session,
      timeZone: "America/New_York",
      day: "2026-03-08",
      roomId: "room-b",
      deltaY: HOUR_HEIGHT,
      hourHeight: HOUR_HEIGHT,
    })
    expect(input).toEqual({
      session,
      roomId: "room-b",
      startLocal: "2026-03-08T10:00:00",
      endLocal: "2026-03-08T11:00:00",
    })
  })

  it("returns null when the slot did not change", () => {
    expect(
      dropToMove({
        session,
        timeZone: "America/New_York",
        day: "2026-03-08",
        roomId: "room-a",
        deltaY: 0,
        hourHeight: HOUR_HEIGHT,
      }),
    ).toBeNull()
  })
})
