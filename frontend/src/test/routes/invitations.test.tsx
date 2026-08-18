import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import {
  createMemoryHistory,
  createRouter,
  RouterProvider,
} from "@tanstack/react-router"
import { fireEvent, render, screen, waitFor } from "@testing-library/react"
import { http, HttpResponse } from "msw"
import { setupServer } from "msw/node"
import { afterAll, afterEach, beforeAll, describe, expect, it } from "vitest"

import type { Event, Invitation, InvitationPage } from "@/lib/api/types"
import { routeTree } from "@/routeTree.gen"

const eventId = "019b0000-0000-7000-8000-000000000001"
const page1Cursor = "opaque-token-page-1"
const midListCursor = "opaque-token-mid-list"

const event: Event = {
  id: eventId,
  name: "DST Fall Back",
  description: "Fixture",
  timeZone: "America/New_York",
  startsAt: "2026-11-01T05:00:00.000Z",
  endsAt: "2026-11-02T05:00:00.000Z",
}

function invite(
  id: string,
  extra: Partial<Invitation> = {},
): Invitation {
  return {
    id,
    eventId,
    role: "attendee",
    status: "pending",
    ...extra,
  }
}

function page(items: Invitation[], nextCursor?: string): InvitationPage {
  return nextCursor ? { items, nextCursor } : { items }
}

const urls: URL[] = []
const historyActions: string[] = []
const server = setupServer()
const originalOffsetHeight = Object.getOwnPropertyDescriptor(
  HTMLElement.prototype,
  "offsetHeight",
)
const originalOffsetWidth = Object.getOwnPropertyDescriptor(
  HTMLElement.prototype,
  "offsetWidth",
)

beforeAll(() => {
  window.scrollTo = () => {}
  server.listen({ onUnhandledRequest: "bypass" })
})
afterEach(() => {
  urls.length = 0
  historyActions.length = 0
  server.resetHandlers()
  if (originalOffsetHeight) {
    Object.defineProperty(HTMLElement.prototype, "offsetHeight", originalOffsetHeight)
  }
  if (originalOffsetWidth) {
    Object.defineProperty(HTMLElement.prototype, "offsetWidth", originalOffsetWidth)
  }
})
afterAll(() => server.close())

function mockViewport() {
  Object.defineProperty(HTMLElement.prototype, "offsetHeight", {
    configurable: true,
    get() {
      if (this.getAttribute?.("data-testid") === "invitation-scroll") return 384
      return 48
    },
  })
  Object.defineProperty(HTMLElement.prototype, "offsetWidth", {
    configurable: true,
    get() {
      return 640
    },
  })
}

function renderRoute(search = "") {
  mockViewport()
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  const history = createMemoryHistory({
    initialEntries: [`/events/${eventId}/invitations${search}`],
  })
  history.subscribe(({ action }) => {
    historyActions.push(action.type)
  })
  const router = createRouter({
    routeTree,
    history,
    context: { queryClient },
    defaultPendingMs: 0,
    defaultPendingMinMs: 0,
    scrollRestoration: false,
  })
  return {
    queryClient,
    history,
    router,
    ...render(
      <QueryClientProvider client={queryClient}>
        <RouterProvider router={router} />
      </QueryClientProvider>,
    ),
  }
}

describe("invitations search state", () => {
  it("sends the URL cursor on the first invitations request", async () => {
    server.use(
      http.get(/\/events\/[^/]+\/invitations/, ({ request }) => {
        const url = new URL(request.url)
        urls.push(url)
        const cursor = url.searchParams.get("cursor")
        if (cursor === midListCursor) {
          return HttpResponse.json(
            page([invite("inv-mid", { email: "mid@example.com" })]),
          )
        }
        return HttpResponse.json(
          page([invite("inv-top", { email: "top@example.com" })]),
        )
      }),
      http.get(/\/events\/[^/]+$/, () => HttpResponse.json(event)),
    )

    renderRoute(`?cursor=${midListCursor}`)

    await waitFor(() => {
      expect(screen.getByText("mid@example.com")).toBeInTheDocument()
    })
    expect(screen.queryByText("top@example.com")).not.toBeInTheDocument()
    expect(urls[0]?.searchParams.get("cursor")).toBe(midListCursor)
    expect(urls[0]?.searchParams.has("status")).toBe(false)
    expect(urls[0]?.searchParams.has("offset")).toBe(false)
  })

  it("writes the filter to the URL and re-queries from the start", async () => {
    server.use(
      http.get(/\/events\/[^/]+\/invitations/, ({ request }) => {
        const url = new URL(request.url)
        urls.push(url)
        return HttpResponse.json(
          page([
            invite("inv-pending", {
              email: "pending@example.com",
              status: "pending",
            }),
            invite("inv-accepted", {
              email: "accepted@example.com",
              status: "accepted",
            }),
          ]),
        )
      }),
      http.get(/\/events\/[^/]+$/, () => HttpResponse.json(event)),
    )

    const { router } = renderRoute(`?cursor=${midListCursor}`)

    await waitFor(() => {
      expect(screen.getByText("pending@example.com")).toBeInTheDocument()
    })
    expect(screen.getByText("accepted@example.com")).toBeInTheDocument()

    fireEvent.change(screen.getByLabelText("Invitation status"), {
      target: { value: "accepted" },
    })

    await waitFor(() => {
      expect(router.state.location.search).toMatchObject({ status: "accepted" })
    })
    expect(router.state.location.search.cursor).toBeUndefined()
    expect(screen.queryByText("pending@example.com")).not.toBeInTheDocument()
    expect(screen.getByText("accepted@example.com")).toBeInTheDocument()

    const afterFilter = urls.filter((url) => url.searchParams.has("cursor") === false)
    expect(afterFilter.length).toBeGreaterThanOrEqual(1)
    expect(urls.every((url) => !url.searchParams.has("status"))).toBe(true)
  })

  it("ignores an invalid status and a garbage cursor without crashing", async () => {
    server.use(
      http.get(/\/events\/[^/]+\/invitations/, ({ request }) => {
        const url = new URL(request.url)
        urls.push(url)
        if (url.searchParams.get("cursor") === "not-a-real-token") {
          return HttpResponse.json(page([]))
        }
        return HttpResponse.json(
          page([invite("inv-1", { email: "one@example.com" })]),
        )
      }),
      http.get(/\/events\/[^/]+$/, () => HttpResponse.json(event)),
    )

    expect(() => renderRoute("?status=nope&cursor=not-a-real-token")).not.toThrow()

    await waitFor(() => {
      expect(screen.getByText("No invitations on this event.")).toBeInTheDocument()
    })
    expect(screen.getByLabelText("Invitation status")).toHaveValue("all")
    expect(urls[0]?.searchParams.get("cursor")).toBe("not-a-real-token")
  })

  it("replaces the URL with a later cursor as pages advance", async () => {
    server.use(
      http.get(/\/events\/[^/]+\/invitations/, ({ request }) => {
        const url = new URL(request.url)
        urls.push(url)
        const cursor = url.searchParams.get("cursor")
        if (!cursor) {
          return HttpResponse.json(
            page(
              [
                invite("inv-1", { email: "one@example.com" }),
                invite("inv-2", { email: "two@example.com" }),
              ],
              page1Cursor,
            ),
          )
        }
        return HttpResponse.json(
          page([invite("inv-3", { email: "three@example.com" })]),
        )
      }),
      http.get(/\/events\/[^/]+$/, () => HttpResponse.json(event)),
    )

    const { history, router } = renderRoute()

    await waitFor(() => {
      expect(screen.getByText("one@example.com")).toBeInTheDocument()
    })
    await waitFor(() => {
      expect(router.state.location.search.cursor).toBe(page1Cursor)
    })
    expect(history.length).toBe(1)
    expect(historyActions.filter((type) => type === "PUSH")).toHaveLength(0)
    expect(historyActions.filter((type) => type === "REPLACE").length).toBeGreaterThanOrEqual(1)
    expect(urls[0]?.searchParams.has("cursor")).toBe(false)
    expect(urls.some((url) => url.searchParams.get("cursor") === page1Cursor)).toBe(true)
  })
})
