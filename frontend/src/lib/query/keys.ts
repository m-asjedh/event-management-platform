export const queryKeys = {
  health: ["health"] as const,
  events: ["events"] as const,
  event: (id: string) => ["events", id] as const,
  rooms: (eventId: string) => ["events", eventId, "rooms"] as const,
  sessions: (eventId: string) => ["events", eventId, "sessions"] as const,
}
