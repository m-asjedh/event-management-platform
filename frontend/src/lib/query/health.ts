import { queryOptions } from "@tanstack/react-query"

import { getJSON } from "@/lib/api/client"
import type { paths } from "@/generated/api"
import { queryKeys } from "@/lib/query/keys"

// Derived from the spec. Do not replace with a handwritten interface.
export type Health =
  paths["/healthz"]["get"]["responses"]["200"]["content"]["application/json"]

export const healthQuery = queryOptions({
  queryKey: queryKeys.health,
  queryFn: () => getJSON<Health>("/healthz"),
})
