import { queryOptions } from "@tanstack/react-query"

import { getJSON } from "@/lib/api/client"
import { isApiError } from "@/lib/api/error"
import type { Me } from "@/lib/api/types"
import { queryKeys } from "@/lib/query/keys"

export const meQuery = queryOptions({
  queryKey: queryKeys.me,
  retry: false,
  queryFn: async (): Promise<Me | null> => {
    try {
      return await getJSON<Me>("/me")
    } catch (error) {
      if (
        isApiError(error) &&
        (error.status === 401 || error.body.code === "UNAUTHENTICATED")
      ) {
        return null
      }
      throw error
    }
  },
})
