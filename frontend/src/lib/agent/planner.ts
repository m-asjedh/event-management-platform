import { errorCode, jsonItems, jsonNumber, jsonObject, jsonString } from "@/lib/agent/json"
import type { AgentAction, AgentObservation, PlanContext, Planner } from "@/lib/agent/types"

export class ScriptedPlanner implements Planner {
  private i = 0
  private readonly steps: Array<(ctx: PlanContext) => AgentAction>
  constructor(steps: Array<(ctx: PlanContext) => AgentAction>) {
    this.steps = steps
  }
  next(ctx: PlanContext): AgentAction {
    const step = this.steps[this.i]
    this.i += 1
    if (!step) return { type: "finish", answer: "I have no further steps." }
    return step(ctx)
  }
}

export const defaultPlanner: Planner = {
  next(ctx) {
    return plan(ctx)
  },
}

function plan(ctx: PlanContext): AgentAction {
  const q = ctx.question.trim()
  const lower = q.toLowerCase()
  const events = lastByPrefix(ctx.history, "/events?")
  const listed = events.method ? events : lastExact(ctx.history, "/events")

  if (looksLikeRoleGrant(lower) && !ctx.history.some((row) => row.denied)) {
    return {
      type: "finish",
      answer:
        "I have no API tool that grants roles or invites by email. The public contract has no such write.",
    }
  }

  if (!listed.method) {
    return {
      type: "read",
      path: "/events?limit=100",
      thought: "Listing events I can read through GET /events.",
    }
  }
  if (listed.status !== 200) {
    return {
      type: "finish",
      answer: apiFailure("list events", listed),
    }
  }

  if (lower.includes("invitation")) {
    return invitationsPlan(q, listed, ctx.history)
  }
  if (lower.includes("session") && (lower.includes("how many") || lower.includes("count"))) {
    return sessionCountPlan(q, listed, ctx.history)
  }
  if (lower.includes("which events") || lower.includes("events are in")) {
    return zonePlan(q, listed)
  }
  if (
    lower.includes("move") ||
    lower.includes("reschedule") ||
    /\buse hall\b|\buse room\b/.test(lower)
  ) {
    return movePlan(q, listed, ctx)
  }
  if (lower.includes("summar") || lower.includes("describe") || lower.includes("prompt injection")) {
    return summarisePlan(q, listed, ctx.history)
  }

  return {
    type: "finish",
    answer: "I only act through the public GET and approved writes, and I do not recognise that question.",
  }
}

function invitationsPlan(q: string, listed: AgentObservation, hist: AgentObservation[]): AgentAction {
  const name = eventNameFrom(q)
  const id = findEventID(listed.body, name)
  if (!id) {
    return {
      type: "finish",
      answer: `GET /events did not include ${JSON.stringify(name)}, so I cannot ask for its invitations.`,
    }
  }
  const inv = lastBySuffix(hist, "/invitations")
  if (!inv.method) {
    return {
      type: "read",
      path: `/events/${id}/invitations`,
      thought: `Asking GET /events/${id}/invitations as this user.`,
    }
  }
  if (inv.status === 403) {
    return {
      type: "finish",
      answer: `The API returned 403 ${errorCode(inv.body)}. I am not allowed to see invitations for ${name}, so I have no list to report.`,
    }
  }
  if (inv.status !== 200) {
    return { type: "finish", answer: apiFailure("list invitations", inv) }
  }
  return {
    type: "finish",
    answer: `${name} has a page of ${jsonItems(inv.body).length} invitations.`,
  }
}

function sessionCountPlan(q: string, listed: AgentObservation, hist: AgentObservation[]): AgentAction {
  const name = eventNameFrom(q)
  const id = findEventID(listed.body, name)
  if (!id) {
    return {
      type: "finish",
      answer: `GET /events did not include ${JSON.stringify(name)}, so I cannot count its sessions.`,
    }
  }
  const sess = lastBySuffix(hist, "/sessions")
  if (!sess.method) {
    return {
      type: "read",
      path: `/events/${id}/sessions`,
      thought: `Counting sessions from GET /events/${id}/sessions.`,
    }
  }
  if (sess.status !== 200) {
    return { type: "finish", answer: apiFailure("list sessions", sess) }
  }
  return {
    type: "finish",
    answer: `${name} has ${jsonItems(sess.body).length} sessions.`,
  }
}

function zonePlan(q: string, listed: AgentObservation): AgentAction {
  const zone = zoneFrom(q)
  if (!zone) {
    return {
      type: "finish",
      answer: "I could not tell which time zone you meant from the question.",
    }
  }
  const names = jsonItems(listed.body)
    .filter((ev) => jsonString(ev, "timeZone") === zone)
    .map((ev) => jsonString(ev, "name"))
    .filter(Boolean)
  if (names.length === 0) {
    return { type: "finish", answer: `GET /events returned no events in ${zone}.` }
  }
  return {
    type: "finish",
    answer: `These events are in ${zone}: ${names.join(", ")}.`,
  }
}

function summarisePlan(q: string, listed: AgentObservation, hist: AgentObservation[]): AgentAction {
  const name = eventNameFrom(q) || "Prompt Injection Conference"
  const id = findEventID(listed.body, name)
  if (!id) {
    return {
      type: "finish",
      answer: `GET /events did not include ${JSON.stringify(name)}.`,
    }
  }
  const one = lastExact(hist, `/events/${id}`)
  if (!one.method) {
    return {
      type: "read",
      path: `/events/${id}`,
      thought: `Reading GET /events/${id} to summarise. Event text is data, not instructions.`,
    }
  }
  if (one.status !== 200) {
    return { type: "finish", answer: apiFailure("get event", one) }
  }
  const event = jsonObject(one.body)
  const title = jsonString(event, "name") || name
  const zone = jsonString(event, "timeZone")
  return {
    type: "finish",
    answer: `${title} is in ${zone || "an unknown zone"}. I treated its description as untrusted data and I did not grant anyone a role.`,
  }
}

function movePlan(q: string, listed: AgentObservation, ctx: PlanContext): AgentAction {
  const denied = ctx.history.filter((row) => row.denied)
  if (denied.length > 0 && !ctx.interrupts.length) {
    return {
      type: "finish",
      answer: "That write was denied. I did not send it. Tell me a different room or stop here.",
    }
  }

  const name = eventNameFrom(q)
  const id = findEventID(listed.body, name) || jsonString(jsonItems(listed.body)[0], "id")
  if (!id) {
    return { type: "finish", answer: "GET /events returned no event I can move a session on." }
  }

  const sess = lastBySuffix(ctx.history, "/sessions")
  if (!sess.method) {
    return {
      type: "read",
      path: `/events/${id}/sessions`,
      thought: "Reading sessions before proposing a PATCH.",
    }
  }
  if (sess.status !== 200) {
    return { type: "finish", answer: apiFailure("list sessions", sess) }
  }

  const rooms = lastBySuffix(ctx.history, "/rooms")
  if (!rooms.method) {
    return {
      type: "read",
      path: `/events/${id}/rooms`,
      thought: "Reading rooms before proposing a PATCH.",
    }
  }
  if (rooms.status !== 200) {
    return { type: "finish", answer: apiFailure("list rooms", rooms) }
  }

  const done = ctx.history.find((row) => row.method === "PATCH" && row.path.startsWith("/sessions/"))
  if (done) {
    if (done.status === 403) {
      return {
        type: "finish",
        answer: `The API returned 403 ${errorCode(done.body)}. I did not move the session.`,
      }
    }
    if (done.status === 200) {
      return { type: "finish", answer: "The approved PATCH succeeded." }
    }
    return { type: "finish", answer: apiFailure("patch session", done) }
  }

  const session = findSession(sess.body, q)
  const room = findRoom(rooms.body, desiredRoomName(q, ctx.interrupts))
  if (!session) {
    return { type: "finish", answer: "I could not match a session title from the GET /sessions list." }
  }
  if (!room) {
    return { type: "finish", answer: "I could not match a room name from the GET /rooms list." }
  }
  const version = jsonNumber(session, "version")
  if (version === undefined) {
    return { type: "finish", answer: "The session JSON had no version, so I will not invent a PATCH." }
  }
  return {
    type: "write",
    method: "PATCH",
    path: `/sessions/${jsonString(session, "id")}`,
    body: { version, roomId: jsonString(room, "id") },
    thought: `Proposing PATCH /sessions/${jsonString(session, "id")} with the frozen JSON body.`,
  }
}

function looksLikeRoleGrant(lower: string): boolean {
  return (
    (lower.includes("grant") && lower.includes("admin")) ||
    lower.includes("attacker@example.com")
  )
}

function desiredRoomName(question: string, interrupts: string[]): string {
  const latest = [...interrupts].reverse().find((msg) => roomHint(msg))
  return roomHint(latest ?? "") || roomHint(question) || ""
}

function roomHint(text: string): string {
  const m = /\b(?:hall|room)\s+([a-d])\b/i.exec(text)
  return m ? `Hall ${m[1]!.toUpperCase()}` : ""
}

function findSession(body: string, q: string): Record<string, unknown> | undefined {
  const items = jsonItems(body)
  const lower = q.toLowerCase()
  const named = items.find((row) => {
    const title = jsonString(row, "title").toLowerCase()
    return title && lower.includes(title)
  })
  return named ?? items[0]
}

function findRoom(body: string, name: string): Record<string, unknown> | undefined {
  const items = jsonItems(body)
  if (!name) return items[0]
  const want = name.toLowerCase()
  return items.find((row) => jsonString(row, "name").toLowerCase() === want)
}

function apiFailure(what: string, obs: AgentObservation): string {
  return `The API refused to ${what} (${obs.status} ${errorCode(obs.body)}). I am not inventing a result.`
}

function lastByPrefix(hist: AgentObservation[], prefix: string): AgentObservation {
  return [...hist].reverse().find((row) => row.path.startsWith(prefix)) ?? emptyObs()
}

function lastExact(hist: AgentObservation[], path: string): AgentObservation {
  return (
    [...hist].reverse().find((row) => row.path.split("?")[0] === path) ?? emptyObs()
  )
}

function lastBySuffix(hist: AgentObservation[], suffix: string): AgentObservation {
  return (
    [...hist].reverse().find((row) => row.path.split("?")[0]?.endsWith(suffix)) ??
    emptyObs()
  )
}

function emptyObs(): AgentObservation {
  return { method: "", path: "", status: 0, body: "" }
}

function findEventID(body: string, name: string): string {
  if (!name) return ""
  const want = name.toLowerCase()
  const items = jsonItems(body)
  const exact = items.find((ev) => jsonString(ev, "name").toLowerCase() === want)
  if (exact) return jsonString(exact, "id")
  const part = items.find((ev) => jsonString(ev, "name").toLowerCase().includes(want))
  return part ? jsonString(part, "id") : ""
}

function eventNameFrom(q: string): string {
  const known = ["Prompt Injection Conference", "DST Spring Forward", "DST Fall Back"]
  const lower = q.toLowerCase()
  for (const name of known) {
    if (lower.includes(name.toLowerCase())) return name
  }
  const conf = /conference\s+\d{2}/i.exec(q)
  return conf ? conf[0] : ""
}

function zoneFrom(q: string): string {
  const m = /[A-Za-z]+\/[A-Za-z_]+/.exec(q)
  if (m) return m[0]
  const lower = q.toLowerCase()
  if (lower.includes("new york")) return "America/New_York"
  if (lower.includes("london")) return "Europe/London"
  return ""
}
