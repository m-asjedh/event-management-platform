import { render, screen, within } from "@testing-library/react"
import { describe, expect, it } from "vitest"

import { ScheduleGrid } from "@/components/schedule/ScheduleGrid"
import type { Event, Room, Session } from "@/lib/api/types"

const eventZone = "America/New_York"

const event: Event = {
  id: "019b0000-0000-7000-8000-000000000001",
  name: "DST Spring Forward",
  description: "Fixture",
  timeZone: eventZone,
  startsAt: "2026-03-07T14:00:00.000Z",
  endsAt: "2026-03-09T22:00:00.000Z",
}

const hallA: Room = {
  id: "room-a",
  eventId: event.id,
  name: "Hall A",
  capacity: 80,
}

const hallB: Room = {
  id: "room-b",
  eventId: event.id,
  name: "Hall B",
  capacity: 80,
}

function session(
  partial: Pick<Session, "id" | "title" | "roomId" | "startsAt" | "endsAt">,
): Session {
  return {
    eventId: event.id,
    description: "",
    version: 1,
    ...partial,
  }
}

// 09:00–10:00 and 09:30–10:30 EDT on 2026-03-08 = 13:00Z / 13:30Z / 14:00Z / 14:30Z.
const clashA = session({
  id: "sess-clash-a",
  title: "Talk Clash A",
  roomId: hallA.id,
  startsAt: "2026-03-08T13:00:00.000Z",
  endsAt: "2026-03-08T14:00:00.000Z",
})

const clashB = session({
  id: "sess-clash-b",
  title: "Talk Clash B",
  roomId: hallA.id,
  startsAt: "2026-03-08T13:30:00.000Z",
  endsAt: "2026-03-08T14:30:00.000Z",
})

const fine = session({
  id: "sess-fine",
  title: "Talk Fine",
  roomId: hallB.id,
  startsAt: "2026-03-08T13:00:00.000Z",
  endsAt: "2026-03-08T14:00:00.000Z",
})

describe("ScheduleGrid", () => {
  it("renders sessions in the right room slots and flags a same-room overlap", () => {
    expect(process.env.TZ).toBe("Pacific/Auckland")

    render(
      <ScheduleGrid
        event={event}
        rooms={[hallA, hallB]}
        sessions={[clashA, clashB, fine]}
        day="2026-03-08"
      />,
    )

    const colA = screen.getByTestId(`room-column-${hallA.id}`)
    const colB = screen.getByTestId(`room-column-${hallB.id}`)

    expect(within(colA).getByText("Talk Clash A")).toBeInTheDocument()
    expect(within(colA).getByText("Talk Clash B")).toBeInTheDocument()
    expect(within(colB).getByText("Talk Fine")).toBeInTheDocument()
    expect(within(colA).queryByText("Talk Fine")).not.toBeInTheDocument()

    const clashAEl = screen.getByRole("article", { name: /Talk Clash A/ })
    const clashBEl = screen.getByRole("article", { name: /Talk Clash B/ })
    const fineEl = screen.getByRole("article", { name: /Talk Fine/ })

    expect(clashAEl).toHaveAttribute("data-conflict", "true")
    expect(clashBEl).toHaveAttribute("data-conflict", "true")
    expect(fineEl).toHaveAttribute("data-conflict", "false")
    expect(within(clashAEl).getByText("Conflict")).toBeInTheDocument()
    expect(within(clashBEl).getByText("Conflict")).toBeInTheDocument()
    expect(within(fineEl).queryByText("Conflict")).not.toBeInTheDocument()
  })

  it("shows event-local times, not the test machine zone", () => {
    expect(process.env.TZ).toBe("Pacific/Auckland")

    render(
      <ScheduleGrid
        event={event}
        rooms={[hallA, hallB]}
        sessions={[clashA, fine]}
        day="2026-03-08"
      />,
    )

    const clashAEl = screen.getByRole("article", { name: /Talk Clash A/ })
    expect(clashAEl).toHaveAttribute("data-start-local", "09:00")
    expect(clashAEl).toHaveAttribute("data-end-local", "10:00")
    expect(within(clashAEl).getByText("09:00–10:00")).toBeInTheDocument()
    expect(screen.queryByText(/02:00/)).not.toBeInTheDocument()
  })
})
