import type { components, paths } from "@/generated/api"

// Aliases of generated schema. Do not replace with handwritten shapes.
export type ErrorBody = components["schemas"]["Error"]
export type Event = components["schemas"]["Event"]
export type Session = components["schemas"]["Session"]
export type Room = components["schemas"]["Room"]
export type EventPage = components["schemas"]["EventPage"]
export type SessionList =
  paths["/events/{eventId}/sessions"]["get"]["responses"]["200"]["content"]["application/json"]
export type RoomList =
  paths["/events/{eventId}/rooms"]["get"]["responses"]["200"]["content"]["application/json"]
