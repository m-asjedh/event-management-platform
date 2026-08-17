import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { useSuspenseQuery } from "@tanstack/react-query"
import { fireEvent, render, screen, waitFor, within } from "@testing-library/react"
import { http, HttpResponse } from "msw"
import { setupServer } from "msw/node"
import { afterAll, afterEach, beforeAll, describe, expect, it } from "vitest"

import { MoveNoticeBanner } from "@/components/schedule/MoveNoticeBanner"
import { ScheduleGrid } from "@/components/schedule/ScheduleGrid"
import type { Event, Room, Session, SessionPatch } from "@/lib/api/types"
import { useMoveSession } from "@/lib/query/moveSession"
import { sessionsQuery } from "@/lib/query/sessions"

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

const talkA: Session = {
  id: "sess-talk-a",
  eventId: event.id,
  title: "Talk A",
  description: "",
  roomId: hallA.id,
  startsAt: "2026-03-08T13:00:00.000Z",
  endsAt: "2026-03-08T14:00:00.000Z",
  version: 1,
}

const patches: SessionPatch[] = []

const server = setupServer()

beforeAll(() => server.listen({ onUnhandledRequest: "bypass" }))
afterEach(() => {
  patches.length = 0
  server.resetHandlers()
})
afterAll(() => server.close())

function Harness() {
  const { data } = useSuspenseQuery(sessionsQuery(event.id))
  const move = useMoveSession({
    eventId: event.id,
    timeZone: event.timeZone,
    rooms: [hallA, hallB],
  })
  const talk = data.items.find((session) => session.id === talkA.id)

  return (
    <>
      <button
        type="button"
        disabled={!talk}
        onClick={() => {
          if (!talk) return
          move.mutate({
            session: talk,
            roomId: hallB.id,
            startLocal: "2026-03-08T10:00:00",
            endLocal: "2026-03-08T11:00:00",
          })
        }}
      >
        Apply move
      </button>
      <MoveNoticeBanner notice={move.notice} onDismiss={move.clearNotice} />
      <ScheduleGrid
        event={event}
        rooms={[hallA, hallB]}
        sessions={data.items}
        day="2026-03-08"
      />
    </>
  )
}

function renderBoard(items: Session[] = [talkA]) {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false, staleTime: Infinity },
      mutations: { retry: false },
    },
  })
  queryClient.setQueryData(sessionsQuery(event.id).queryKey, { items })
  const view = render(
    <QueryClientProvider client={queryClient}>
      <Harness />
    </QueryClientProvider>,
  )
  return { ...view, queryClient }
}

function article(title: string) {
  return screen.getByRole("article", { name: new RegExp(title) })
}

describe("drag-to-reschedule mutation", () => {
  it("confirms a successful move and sends the bumped version on the next edit", async () => {
    server.use(
      http.patch(/\/sessions\/[^/]+$/, async ({ request }) => {
        const body = (await request.json()) as SessionPatch
        patches.push(body)
        return HttpResponse.json({
          ...talkA,
          roomId: body.roomId ?? null,
          startsAt: "2026-03-08T14:00:00.000Z",
          endsAt: "2026-03-08T15:00:00.000Z",
          version: body.version + 1,
        })
      }),
    )

    renderBoard()
    fireEvent.click(screen.getByRole("button", { name: "Apply move" }))

    await waitFor(() => {
      expect(article("Talk A")).toHaveAttribute("data-room-id", hallB.id)
      expect(article("Talk A")).toHaveAttribute("data-start-local", "10:00")
      expect(article("Talk A")).toHaveAttribute("data-version", "2")
    })
    expect(within(screen.getByTestId(`room-column-${hallA.id}`)).queryByText("Talk A")).toBeNull()
    expect(patches[0]?.version).toBe(1)

    fireEvent.click(screen.getByRole("button", { name: "Apply move" }))
    await waitFor(() => expect(patches).toHaveLength(2))
    expect(patches[1]?.version).toBe(2)
    await waitFor(() => expect(article("Talk A")).toHaveAttribute("data-version", "3"))
  })

  it("rolls back a ROOM_CONFLICT and keeps the intended slot in the message", async () => {
    server.use(
      http.patch(/\/sessions\/[^/]+$/, async ({ request }) => {
        patches.push((await request.json()) as SessionPatch)
        return HttpResponse.json(
          {
            code: "ROOM_CONFLICT",
            reason: "room is taken",
            conflict: {
              conflictingSessionId: "sess-other",
              conflictingTitle: "Talk Occupied",
            },
          },
          { status: 409 },
        )
      }),
    )

    renderBoard()
    fireEvent.click(screen.getByRole("button", { name: "Apply move" }))

    await waitFor(() => {
      expect(screen.getByTestId("move-notice")).toHaveAttribute(
        "data-notice-code",
        "ROOM_CONFLICT",
      )
    })
    expect(screen.getByTestId("move-notice")).toHaveTextContent("Hall B at 10:00 is taken")
    expect(screen.getByTestId("move-notice")).toHaveTextContent("Talk Occupied")

    const block = article("Talk A")
    expect(block).toHaveAttribute("data-room-id", hallA.id)
    expect(block).toHaveAttribute("data-start-local", "09:00")
    expect(within(screen.getByTestId(`room-column-${hallB.id}`)).queryByText("Talk A")).toBeNull()
    expect(within(screen.getByTestId(`room-column-${hallA.id}`)).getByText("Talk A")).toBeInTheDocument()
  })

  it("reconciles STALE_VERSION from the error body, not the optimistic slot", async () => {
    const current: Session = {
      ...talkA,
      version: 7,
      startsAt: "2026-03-08T15:00:00.000Z",
      endsAt: "2026-03-08T16:00:00.000Z",
      roomId: hallA.id,
    }
    server.use(
      http.patch(/\/sessions\/[^/]+$/, async ({ request }) => {
        patches.push((await request.json()) as SessionPatch)
        return HttpResponse.json(
          {
            code: "STALE_VERSION",
            reason: "version has changed",
            conflict: {
              currentVersion: 7,
              currentState: current,
            },
          },
          { status: 409 },
        )
      }),
    )

    renderBoard()
    fireEvent.click(screen.getByRole("button", { name: "Apply move" }))

    await waitFor(() => {
      expect(screen.getByTestId("move-notice")).toHaveAttribute(
        "data-notice-code",
        "STALE_VERSION",
      )
    })
    expect(screen.getByTestId("move-notice")).toHaveTextContent("schedule changed")

    const block = article("Talk A")
    expect(block).toHaveAttribute("data-version", "7")
    expect(block).toHaveAttribute("data-start-local", "11:00")
    expect(block).toHaveAttribute("data-room-id", hallA.id)
    expect(within(screen.getByTestId(`room-column-${hallB.id}`)).queryByText("Talk A")).toBeNull()
    expect(screen.queryByText("10:00–11:00")).not.toBeInTheDocument()
  })

  it("rolls back FORBIDDEN without offering a retry of the same write", async () => {
    server.use(
      http.patch(/\/sessions\/[^/]+$/, () =>
        HttpResponse.json(
          { code: "FORBIDDEN", reason: "not allowed" },
          { status: 403 },
        ),
      ),
    )

    renderBoard()
    fireEvent.click(screen.getByRole("button", { name: "Apply move" }))

    await waitFor(() => {
      expect(screen.getByTestId("move-notice")).toHaveAttribute("data-notice-code", "FORBIDDEN")
    })
    expect(screen.getByTestId("move-notice")).toHaveTextContent("You can't move this session")
    expect(article("Talk A")).toHaveAttribute("data-room-id", hallA.id)
    expect(article("Talk A")).toHaveAttribute("data-start-local", "09:00")
  })

  it("rolls back VALIDATION_ERROR and surfaces the field reason", async () => {
    server.use(
      http.patch(/\/sessions\/[^/]+$/, () =>
        HttpResponse.json(
          {
            code: "VALIDATION_ERROR",
            reason: "nonexistent local time",
            fieldErrors: [{ field: "startsAt", reason: "nonexistent local time" }],
          },
          { status: 400 },
        ),
      ),
    )

    renderBoard()
    fireEvent.click(screen.getByRole("button", { name: "Apply move" }))

    await waitFor(() => {
      expect(screen.getByTestId("move-notice")).toHaveAttribute(
        "data-notice-code",
        "VALIDATION_ERROR",
      )
    })
    expect(screen.getByTestId("move-notice")).toHaveTextContent("startsAt: nonexistent local time")
    expect(article("Talk A")).toHaveAttribute("data-start-local", "09:00")
  })
})
