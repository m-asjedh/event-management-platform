import { infiniteQueryOptions } from "@tanstack/react-query"

import { getJSON } from "@/lib/api/client"
import type { InvitationCursor, InvitationPage, InvitationStatus } from "@/lib/api/types"
import { queryKeys } from "@/lib/query/keys"

/** Within the API max of 100. Not the whole 50k. */
export const INVITATION_PAGE_SIZE = 50

export function invitationsQuery(
  eventId: string,
  opts: {
    cursor?: InvitationCursor | null
    status?: InvitationStatus
  } = {},
) {
  const startCursor = opts.cursor ?? null
  return infiniteQueryOptions({
    queryKey: queryKeys.invitations(eventId, opts.status),
    queryFn: ({ pageParam }: { pageParam: InvitationCursor | null }) => {
      const params = new URLSearchParams({ limit: String(INVITATION_PAGE_SIZE) })
      if (pageParam) params.set("cursor", pageParam)
      return getJSON<InvitationPage>(`/events/${eventId}/invitations?${params}`)
    },
    initialPageParam: startCursor,
    getNextPageParam: (lastPage) => lastPage.nextCursor || undefined,
  })
}
