import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { render, screen, waitFor } from "@testing-library/react"
import { http, HttpResponse } from "msw"
import { setupServer } from "msw/node"
import { afterAll, afterEach, beforeAll, describe, expect, it } from "vitest"

import { InvitationList } from "@/components/invitations/InvitationList"
import type { Invitation, InvitationPage } from "@/lib/api/types"
import { invitationsQuery } from "@/lib/query/invitations"

const eventId = "019b0000-0000-7000-8000-000000000001"
const page1Cursor = "opaque-token-page-1"

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
const server = setupServer()
const originalOffsetHeight = Object.getOwnPropertyDescriptor(
  HTMLElement.prototype,
  "offsetHeight",
)
const originalOffsetWidth = Object.getOwnPropertyDescriptor(
  HTMLElement.prototype,
  "offsetWidth",
)

beforeAll(() => server.listen({ onUnhandledRequest: "bypass" }))
afterEach(() => {
  urls.length = 0
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

function renderList(client?: QueryClient) {
  const queryClient =
    client ??
    new QueryClient({
      defaultOptions: {
        queries: { retry: false },
      },
    })
  mockViewport()
  return {
    queryClient,
    ...render(
      <QueryClientProvider client={queryClient}>
        <InvitationList eventId={eventId} />
      </QueryClientProvider>,
    ),
  }
}

describe("InvitationList", () => {
  it("renders the first page and requests the next page with the opaque cursor, not an offset", async () => {
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
    )

    renderList()

    await waitFor(() => {
      expect(screen.getByText("one@example.com")).toBeInTheDocument()
    })
    expect(screen.getByText("two@example.com")).toBeInTheDocument()

    await waitFor(() => expect(urls.length).toBeGreaterThanOrEqual(2))
    expect(urls[0]?.searchParams.has("cursor")).toBe(false)
    expect(urls[0]?.searchParams.has("offset")).toBe(false)
    expect(urls[1]?.searchParams.get("cursor")).toBe(page1Cursor)
    expect(urls[1]?.searchParams.has("offset")).toBe(false)
    expect(urls.every((url) => !url.searchParams.has("offset"))).toBe(true)
  })

  it("stops fetching when nextCursor is absent", async () => {
    server.use(
      http.get(/\/events\/[^/]+\/invitations/, ({ request }) => {
        urls.push(new URL(request.url))
        return HttpResponse.json(
          page([invite("inv-end", { email: "last@example.com" })]),
        )
      }),
    )

    renderList()
    await waitFor(() => {
      expect(screen.getByText("last@example.com")).toBeInTheDocument()
    })
    await waitFor(() => {
      expect(screen.getByText("End of invitations.")).toBeInTheDocument()
    })
    expect(screen.queryByRole("status", { name: /loading more/i })).not.toBeInTheDocument()
    expect(urls).toHaveLength(1)
    expect(urls[0]?.searchParams.has("cursor")).toBe(false)
  })

  it("surfaces a typed FORBIDDEN instead of a blank list", async () => {
    server.use(
      http.get(/\/events\/[^/]+\/invitations/, () =>
        HttpResponse.json(
          { code: "FORBIDDEN", reason: "not allowed" },
          { status: 403 },
        ),
      ),
    )

    renderList()
    await waitFor(() => {
      expect(screen.getByRole("heading", { name: "FORBIDDEN" })).toBeInTheDocument()
    })
    expect(screen.getByText("not allowed")).toBeInTheDocument()
    expect(screen.queryByTestId("invitation-row")).not.toBeInTheDocument()
  })

  it("renders rows that omit email", async () => {
    server.use(
      http.get(/\/events\/[^/]+\/invitations/, () =>
        HttpResponse.json(page([invite("inv-no-email")])),
      ),
    )

    renderList()
    await waitFor(() => {
      expect(screen.getByText("Email hidden")).toBeInTheDocument()
    })
    expect(screen.getByText("attendee")).toBeInTheDocument()
    expect(screen.queryByText("@")).not.toBeInTheDocument()
  })

  it("keeps only a bounded number of row nodes in the DOM after many pages load", () => {
    const items = Array.from({ length: 150 }, (_, i) =>
      invite(`inv-${String(i).padStart(3, "0")}`, {
        email: `user${i}@example.com`,
      }),
    )
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false, staleTime: Infinity } },
    })
    queryClient.setQueryData(invitationsQuery(eventId).queryKey, {
      pages: [
        page(items.slice(0, 50), "c1"),
        page(items.slice(50, 100), "c2"),
        page(items.slice(100), undefined),
      ],
      pageParams: [null, "c1", "c2"],
    })

    renderList(queryClient)

    const rows = screen.getAllByTestId("invitation-row")
    expect(rows.length).toBeGreaterThan(0)
    expect(rows.length).toBeLessThan(40)
    expect(rows.length).toBeLessThan(items.length)
  })
})
