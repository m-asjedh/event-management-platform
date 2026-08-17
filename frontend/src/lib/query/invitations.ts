import { infiniteQueryOptions } from "@tanstack/react-query"

import { getJSON } from "@/lib/api/client"
import type { InvitationPage } from "@/lib/api/types"
import { queryKeys } from "@/lib/query/keys"

/** Within the API max of 100. Not the whole 50k. */
export const INVITATION_PAGE_SIZE = 50

export function invitationsQuery(eventId: string) {
  return infiniteQueryOptions({
    queryKey: queryKeys.invitations(eventId),
    queryFn: ({ pageParam }: { pageParam: string | null }) => {
      const params = new URLSearchParams({ limit: String(INVITATION_PAGE_SIZE) })
      if (pageParam) params.set("cursor", pageParam)
      return getJSON<InvitationPage>(`/events/${eventId}/invitations?${params}`)
    },
    initialPageParam: null as string | null,
    getNextPageParam: (lastPage) => lastPage.nextCursor || undefined,
  })
}
