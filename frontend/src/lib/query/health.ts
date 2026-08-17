import { queryKeys } from "@/lib/query/keys"
import { getJSON } from "@/lib/api/client"
import type { paths } from "@/generated/api"

// Derived from the spec. Do not replace with a handwritten interface.
export type Health =
  paths["/healthz"]["get"]["responses"]["200"]["content"]["application/json"]

export const healthQuery = {
  queryKey: queryKeys.health,
  queryFn: () => getJSON<Health>("/healthz"),
}
