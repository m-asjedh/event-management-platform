import type { QueryClient } from "@tanstack/react-query"

import type { Me } from "@/lib/api/types"
import { queryKeys } from "@/lib/query/keys"

export const AUTH_SIGN_IN_PATH = "/api/auth/sign-in/email"
export const AUTH_SIGN_OUT_PATH = "/api/auth/sign-out"

export async function postSignOut(): Promise<void> {
  await fetch(AUTH_SIGN_OUT_PATH, {
    method: "POST",
    credentials: "include",
  })
}

export async function postSignIn(
  email: string,
  password: string,
): Promise<Response> {
  return fetch(AUTH_SIGN_IN_PATH, {
    method: "POST",
    credentials: "include",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email, password }),
  })
}

/** Drop cached API data so the next user cannot see the previous session. */
export async function clearClientSession(
  queryClient: QueryClient,
  me: Me | null = null,
) {
  await queryClient.cancelQueries()
  queryClient.setQueryData(queryKeys.me, me)
  queryClient.removeQueries({
    predicate: (query) => query.queryKey[0] !== "me",
  })
}
