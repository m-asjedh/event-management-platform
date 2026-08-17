import { queryOptions } from "@tanstack/react-query"

import { getJSON } from "@/lib/api/client"
import type { SessionList } from "@/lib/api/types"
import { queryKeys } from "@/lib/query/keys"

export function sessionsQuery(eventId: string) {
  return queryOptions({
    queryKey: queryKeys.sessions(eventId),
    queryFn: () => getJSON<SessionList>(`/events/${eventId}/sessions`),
  })
}
