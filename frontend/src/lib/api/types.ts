import type { components, paths } from "@/generated/api"

// Aliases of generated schema. Do not replace with handwritten shapes.
export type ErrorBody = components["schemas"]["Error"]
export type Event = components["schemas"]["Event"]
export type Session = components["schemas"]["Session"]
export type Room = components["schemas"]["Room"]
export type EventPage = components["schemas"]["EventPage"]
export type SessionPatch = components["schemas"]["SessionPatch"]
export type SessionList =
  paths["/events/{eventId}/sessions"]["get"]["responses"]["200"]["content"]["application/json"]
export type RoomList =
  paths["/events/{eventId}/rooms"]["get"]["responses"]["200"]["content"]["application/json"]
export type PatchedSession =
  paths["/sessions/{id}"]["patch"]["responses"]["200"]["content"]["application/json"]
export type Invitation = components["schemas"]["Invitation"]
export type InvitationPage = components["schemas"]["InvitationPage"]
export type InvitationStatus = Invitation["status"]
export type InvitationCursor = NonNullable<
  NonNullable<
    paths["/events/{eventId}/invitations"]["get"]["parameters"]["query"]
  >["cursor"]
>
