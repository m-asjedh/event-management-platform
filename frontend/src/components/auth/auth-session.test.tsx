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

import type { Me } from "@/lib/api/types"
import { queryKeys } from "@/lib/query/keys"
import { routeTree } from "@/routeTree.gen"

const admin: Me = {
  id: "user-admin",
  name: "Seed Admin",
  email: "seed.admin@example.com",
}

const attendee: Me = {
  id: "user-attendee",
  name: "Seed Attendee",
  email: "seed.attendee@example.com",
}

type Session = "admin" | "attendee" | null

let session: Session = "admin"
const signOutUrls: string[] = []
const signInEmails: string[] = []

const server = setupServer()

beforeAll(() => {
  window.scrollTo = () => {}
  server.listen({ onUnhandledRequest: "bypass" })
})
afterEach(() => {
  session = "admin"
  signOutUrls.length = 0
  signInEmails.length = 0
  server.resetHandlers()
})
afterAll(() => server.close())

function pathnameIs(request: Request, path: string) {
  return new URL(request.url).pathname === path
}

function installAuthHandlers() {
  server.use(
    http.get(({ request }) => pathnameIs(request, "/me"), () => {
      if (!session) {
        return HttpResponse.json(
          { code: "UNAUTHENTICATED", reason: "no session" },
          { status: 401 },
        )
      }
      return HttpResponse.json(session === "admin" ? admin : attendee)
    }),
    http.get(({ request }) => pathnameIs(request, "/healthz"), () =>
      HttpResponse.json({ status: "ok" }),
    ),
    http.get(({ request }) => pathnameIs(request, "/events"), () => {
      if (!session) {
        return HttpResponse.json(
          { code: "UNAUTHENTICATED", reason: "no session" },
          { status: 401 },
        )
      }
      return HttpResponse.json({ items: [] })
    }),
    http.post(({ request }) => pathnameIs(request, "/api/auth/sign-out"), ({ request }) => {
      signOutUrls.push(new URL(request.url).pathname)
      session = null
      return HttpResponse.json({ success: true })
    }),
    http.post(
      ({ request }) => pathnameIs(request, "/api/auth/sign-in/email"),
      async ({ request }) => {
        const body = (await request.json()) as { email: string }
        signInEmails.push(body.email)
        if (body.email === admin.email) session = "admin"
        else if (body.email === attendee.email) session = "attendee"
        else {
          return HttpResponse.json({ message: "invalid password" }, { status: 401 })
        }
        return HttpResponse.json({ user: { email: body.email } })
      },
    ),
  )
}

function renderHome() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  const router = createRouter({
    routeTree,
    history: createMemoryHistory({ initialEntries: ["/"] }),
    context: { queryClient },
    defaultPendingMs: 0,
    defaultPendingMinMs: 0,
    scrollRestoration: false,
  })
  return {
    queryClient,
    router,
    ...render(
      <QueryClientProvider client={queryClient}>
        <RouterProvider router={router} />
      </QueryClientProvider>,
    ),
  }
}

describe("auth session", () => {
  it("shows Sign out only while GET /me returns a user", async () => {
    installAuthHandlers()
    renderHome()

    await waitFor(() => {
      expect(screen.getByText(admin.email)).toBeInTheDocument()
    })
    expect(screen.getByRole("button", { name: "Sign out" })).toBeInTheDocument()
  })

  it("posts sign-out, clears the current-user query, and returns to sign-in", async () => {
    installAuthHandlers()
    const { queryClient } = renderHome()

    await waitFor(() => {
      expect(screen.getByRole("button", { name: "Sign out" })).toBeInTheDocument()
    })

    fireEvent.click(screen.getByRole("button", { name: "Sign out" }))

    await waitFor(() => {
      expect(signOutUrls).toEqual(["/api/auth/sign-out"])
    })
    await waitFor(() => {
      expect(queryClient.getQueryData(queryKeys.me)).toBeNull()
    })
    await waitFor(() => {
      expect(screen.queryByRole("button", { name: "Sign out" })).not.toBeInTheDocument()
    })
    expect(screen.getByRole("heading", { name: "UNAUTHENTICATED" })).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "Sign in" })).toBeInTheDocument()
  })

  it("signs out the current session before signing in as someone else", async () => {
    installAuthHandlers()
    const { queryClient } = renderHome()

    await waitFor(() => {
      expect(screen.getByText(admin.email)).toBeInTheDocument()
    })

    fireEvent.click(screen.getByText("Sign in as someone else"))
    fireEvent.change(screen.getByLabelText("Email"), {
      target: { value: attendee.email },
    })
    fireEvent.click(screen.getByRole("button", { name: "Sign in" }))

    await waitFor(() => {
      expect(signOutUrls).toEqual(["/api/auth/sign-out"])
    })
    await waitFor(() => {
      expect(signInEmails).toEqual([attendee.email])
    })
    await waitFor(() => {
      expect(queryClient.getQueryData(queryKeys.me)).toEqual(attendee)
      expect(screen.getByText(attendee.email)).toBeInTheDocument()
    })
    expect(screen.queryByText(admin.email)).not.toBeInTheDocument()
    expect(screen.getByRole("button", { name: "Sign out" })).toBeInTheDocument()
  })
})
