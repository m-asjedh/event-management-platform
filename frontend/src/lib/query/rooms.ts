import { queryOptions } from "@tanstack/react-query"

import { getJSON } from "@/lib/api/client"
import type { RoomList } from "@/lib/api/types"
import { queryKeys } from "@/lib/query/keys"

export function roomsQuery(eventId: string) {
  return queryOptions({
    queryKey: queryKeys.rooms(eventId),
    queryFn: () => getJSON<RoomList>(`/events/${eventId}/rooms`),
  })
}
