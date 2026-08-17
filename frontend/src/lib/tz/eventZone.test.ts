import { describe, expect, it } from "vitest"

import {
  formatEventTime,
  instantToYmd,
  zonedParts,
} from "@/lib/tz/eventZone"

describe("event zone wall clock", () => {
  // vitest.config sets TZ=Pacific/Auckland. 2026-03-08T13:00:00Z is 02:00 on
  // 9 Mar there, and 09:00 America/New_York (EDT, spring-forward Sunday).
  const instant = "2026-03-08T13:00:00.000Z"
  const eventZone = "America/New_York"

  it("formats the event-local time, not the test machine zone", () => {
    expect(process.env.TZ).toBe("Pacific/Auckland")
    expect(formatEventTime(instant, eventZone)).toBe("09:00")
    expect(instantToYmd(instant, eventZone)).toBe("2026-03-08")

    const machine = zonedParts(instant, "Pacific/Auckland")
    expect(
      `${machine.hour.toString().padStart(2, "0")}:${machine.minute.toString().padStart(2, "0")}`,
    ).toBe("02:00")
    expect(formatEventTime(instant, eventZone)).not.toBe("02:00")
  })
})
