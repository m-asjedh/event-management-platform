import { useInfiniteQuery } from "@tanstack/react-query"
import { useVirtualizer } from "@tanstack/react-virtual"
import { useEffect, useRef } from "react"

import { ApiErrorView } from "@/components/api/ApiErrorView"
import { InvitationRow } from "@/components/invitations/InvitationRow"
import type { InvitationCursor, InvitationStatus } from "@/lib/api/types"
import { invitationsQuery } from "@/lib/query/invitations"

export const INVITATION_ROW_HEIGHT = 48
export const INVITATION_VIEWPORT_HEIGHT = 384

const CURSOR_URL_DEBOUNCE_MS = 200

export function InvitationList({
  eventId,
  cursor = null,
  status,
  onStartCursorChange,
}: {
  eventId: string
  cursor?: InvitationCursor | null
  status?: InvitationStatus
  onStartCursorChange?: (next: InvitationCursor) => void
}) {
  const seedRef = useRef({ status, cursor })
  if (seedRef.current.status !== status) {
    seedRef.current = { status, cursor }
  }

  const {
    data,
    isPending,
    isError,
    error,
    hasNextPage,
    isFetchingNextPage,
    fetchNextPage,
  } = useInfiniteQuery(
    invitationsQuery(eventId, { cursor: seedRef.current.cursor, status }),
  )
  const parentRef = useRef<HTMLDivElement>(null)
  const loaded = data?.pages.flatMap((page) => page.items) ?? []
  const items = status ? loaded.filter((row) => row.status === status) : loaded

  const virtualizer = useVirtualizer({
    count: items.length,
    getScrollElement: () => parentRef.current,
    estimateSize: () => INVITATION_ROW_HEIGHT,
    overscan: 6,
    getItemKey: (index) => items[index]?.id ?? index,
    initialRect: { width: 640, height: INVITATION_VIEWPORT_HEIGHT },
  })

  const virtualRows = virtualizer.getVirtualItems()
  const lastVirtualIndex = virtualRows.at(-1)?.index

  useEffect(() => {
    if (lastVirtualIndex === undefined) return
    if (!hasNextPage || isFetchingNextPage) return
    if (lastVirtualIndex >= items.length - 8) {
      void fetchNextPage()
    }
  }, [lastVirtualIndex, items.length, hasNextPage, isFetchingNextPage, fetchNextPage])

  const advancedCursor = advancedPageCursor(data?.pageParams)

  useEffect(() => {
    if (!onStartCursorChange || !advancedCursor) return
    const timer = window.setTimeout(() => {
      onStartCursorChange(advancedCursor)
    }, CURSOR_URL_DEBOUNCE_MS)
    return () => window.clearTimeout(timer)
  }, [advancedCursor, onStartCursorChange])

  if (isPending) {
    return <p className="text-neutral-600">Loading invitations…</p>
  }

  if (isError) {
    return <ApiErrorView error={error} />
  }

  if (loaded.length === 0) {
    return <p className="text-neutral-700">No invitations on this event.</p>
  }

  if (items.length === 0) {
    return (
      <p className="text-neutral-700">
        No {status} invitations in the loaded pages.
      </p>
    )
  }

  return (
    <div>
      <div
        ref={parentRef}
        data-testid="invitation-scroll"
        className="overflow-auto rounded-md border"
        style={{ height: INVITATION_VIEWPORT_HEIGHT }}
      >
        <div
          className="relative w-full"
          style={{ height: virtualizer.getTotalSize() }}
        >
          {virtualRows.map((row) => {
            const invitation = items[row.index]
            if (!invitation) return null
            return (
              <div
                key={invitation.id}
                data-index={row.index}
                className="absolute top-0 left-0 w-full"
                style={{
                  height: row.size,
                  transform: `translateY(${row.start}px)`,
                }}
              >
                <InvitationRow invitation={invitation} />
              </div>
            )
          })}
        </div>
      </div>
      {isFetchingNextPage ? (
        <p role="status" className="mt-2 text-sm text-neutral-600">
          Loading more…
        </p>
      ) : null}
      {!hasNextPage ? (
        <p className="mt-2 text-sm text-neutral-500">End of invitations.</p>
      ) : null}
    </div>
  )
}

function advancedPageCursor(
  pageParams: readonly unknown[] | undefined,
): InvitationCursor | undefined {
  if (!pageParams || pageParams.length < 2) return undefined
  const latest = pageParams[pageParams.length - 1]
  return typeof latest === "string" && latest.length > 0 ? latest : undefined
}
