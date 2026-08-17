import { queryOptions } from "@tanstack/react-query"

import { getJSON } from "@/lib/api/client"
import type { Event, EventPage } from "@/lib/api/types"
import { queryKeys } from "@/lib/query/keys"

export const eventsQuery = queryOptions({
  queryKey: queryKeys.events,
  queryFn: () => getJSON<EventPage>("/events?limit=50"),
})

export function eventQuery(id: string) {
  return queryOptions({
    queryKey: queryKeys.event(id),
    queryFn: () => getJSON<Event>(`/events/${id}`),
  })
}
