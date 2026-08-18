import type { InvitationCursor, InvitationStatus } from "@/lib/api/types"

export const INVITATION_STATUSES = [
  "pending",
  "accepted",
  "declined",
  "revoked",
] as const satisfies readonly InvitationStatus[]

export type InvitationsSearch = {
  status?: InvitationStatus
  cursor?: InvitationCursor
}

export function isInvitationStatus(value: unknown): value is InvitationStatus {
  return (
    typeof value === "string" &&
    (INVITATION_STATUSES as readonly string[]).includes(value)
  )
}

/** Invalid or missing params become the default. Never throws. */
export function validateInvitationsSearch(
  search: Record<string, unknown>,
): InvitationsSearch {
  const status = isInvitationStatus(search.status) ? search.status : undefined
  const cursor =
    typeof search.cursor === "string" && search.cursor.length > 0
      ? search.cursor
      : undefined
  return { status, cursor }
}
