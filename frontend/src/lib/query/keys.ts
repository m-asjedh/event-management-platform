import type { InvitationStatus } from "@/lib/api/types"

export const queryKeys = {
  health: ["health"] as const,
  events: ["events"] as const,
  event: (id: string) => ["events", id] as const,
  rooms: (eventId: string) => ["events", eventId, "rooms"] as const,
  sessions: (eventId: string) => ["events", eventId, "sessions"] as const,
  invitations: (eventId: string, status?: InvitationStatus) =>
    status
      ? (["events", eventId, "invitations", status] as const)
      : (["events", eventId, "invitations"] as const),
}
