import type { QueryClient } from "@tanstack/react-query"
import { useQuery } from "@tanstack/react-query"
import { Link, Outlet, createRootRouteWithContext } from "@tanstack/react-router"

import { SignOutButton } from "@/components/auth/SignOutButton"
import { meQuery } from "@/lib/query/me"

export const Route = createRootRouteWithContext<{ queryClient: QueryClient }>()({
  component: AppShell,
})

function AppShell() {
  const { data: me } = useQuery(meQuery)

  return (
    <div className="min-h-svh bg-neutral-50 text-neutral-900">
      <header className="sticky top-0 z-30 border-b border-neutral-200 bg-white/90 backdrop-blur">
        <div className="mx-auto flex h-12 max-w-6xl items-center gap-5 px-6">
          <Link
            to="/"
            activeOptions={{ exact: true }}
            className="text-sm text-neutral-600 hover:text-neutral-900"
            activeProps={{ className: "font-semibold text-neutral-900" }}
          >
            Events
          </Link>
          <Link
            to="/agent"
            className="text-sm text-neutral-600 hover:text-neutral-900"
            activeProps={{ className: "font-semibold text-neutral-900" }}
          >
            Agent
          </Link>
          {me ? (
            <div className="ml-auto flex min-w-0 items-center gap-3">
              <span className="truncate text-xs text-neutral-500">{me.email}</span>
              <SignOutButton />
            </div>
          ) : null}
        </div>
      </header>
      <Outlet />
    </div>
  )
}
