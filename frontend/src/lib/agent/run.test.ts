import { describe, expect, it } from "vitest"

import { isPrivilegedWrite } from "@/lib/agent/allowlist"
import { executeApprovedWrite, freezeProposal } from "@/lib/agent/gate"
import type { AgentHttp } from "@/lib/agent/http"
import { ScriptedPlanner } from "@/lib/agent/planner"
import { createAgentRun } from "@/lib/agent/run"
import { SEEDED_INJECTION_TEXT, type AgentEvent, type AgentStatus } from "@/lib/agent/types"

const sessionPatch = { version: 1, roomId: "room-b" }
const patchPath = "/sessions/sess-1"

function recordingHttp(opts?: {
  get?: (path: string) => { status: number; body: string }
  write?: (req: { method: string; path: string; bodyText: string }) => {
    status: number
    body: string
  }
}) {
  const reads: string[] = []
  const writes: Array<{ method: string; path: string; bodyText: string }> = []
  const http: AgentHttp = {
    async get(path) {
      reads.push(path)
      return opts?.get?.(path) ?? { status: 200, body: `{"items":[]}` }
    },
    async sendWrite(req) {
      writes.push(req)
      return opts?.write?.(req) ?? { status: 200, body: `{"id":"sess-1"}` }
    },
  }
  return { http, reads, writes }
}

function collect() {
  const events: AgentEvent[] = []
  const statuses: AgentStatus[] = []
  return {
    events,
    statuses,
    onEvent: (event: AgentEvent) => events.push(event),
    onStatus: (status: AgentStatus) => statuses.push(status),
  }
}

async function waitFor(
  run: ReturnType<typeof createAgentRun>,
  status: AgentStatus,
) {
  for (let i = 0; i < 50; i++) {
    if (run.status() === status) return
    await new Promise((r) => setTimeout(r, 0))
  }
  throw new Error(`timed out waiting for ${status}, have ${run.status()}`)
}

describe("agent write gate", () => {
  it("sends exactly the approved JSON and nothing else", async () => {
    const rec = recordingHttp()
    const bag = collect()
    const run = createAgentRun({
      planner: new ScriptedPlanner([
        () => ({
          type: "write",
          method: "PATCH",
          path: patchPath,
          body: sessionPatch,
          thought: "propose move",
        }),
        () => ({ type: "finish", answer: "The approved PATCH succeeded." }),
      ]),
      http: rec.http,
      onEvent: bag.onEvent,
      onStatus: bag.onStatus,
    })
    const done = run.start("move the session")
    await waitFor(run, "awaiting_approval")
    const proposal = run.pending()
    expect(proposal?.method).toBe("PATCH")
    expect(proposal?.path).toBe(patchPath)
    expect(proposal?.bodyText).toBe(JSON.stringify(sessionPatch))
    expect(rec.writes).toEqual([])
    run.approve()
    await done
    expect(rec.writes).toEqual([
      { method: "PATCH", path: patchPath, bodyText: JSON.stringify(sessionPatch) },
    ])
    expect(bag.events.some((e) => e.type === "completed" && e.answer.includes("succeeded"))).toBe(
      true,
    )
  })

  it("does not send a denied write and then continues", async () => {
    const rec = recordingHttp({
      get: () => ({ status: 200, body: `{"items":[{"id":"e1","name":"DST Spring Forward"}]}` }),
    })
    const bag = collect()
    const run = createAgentRun({
      planner: new ScriptedPlanner([
        () => ({
          type: "write",
          method: "PATCH",
          path: patchPath,
          body: sessionPatch,
          thought: "propose",
        }),
        () => ({
          type: "read",
          path: "/events?limit=100",
          thought: "replan after deny",
        }),
        () => ({ type: "finish", answer: "That write was denied. I listed events instead." }),
      ]),
      http: rec.http,
      onEvent: bag.onEvent,
    })
    const done = run.start("move the session")
    await waitFor(run, "awaiting_approval")
    run.deny()
    await done
    expect(rec.writes).toEqual([])
    expect(rec.reads).toEqual(["/events?limit=100"])
    expect(bag.events.some((e) => e.type === "denied")).toBe(true)
    expect(bag.events.some((e) => e.type === "completed")).toBe(true)
  })

  it("reports an API 403 and does not claim the write succeeded", async () => {
    const rec = recordingHttp({
      write: () => ({
        status: 403,
        body: `{"code":"FORBIDDEN","reason":"not allowed"}`,
      }),
    })
    const bag = collect()
    const run = createAgentRun({
      planner: new ScriptedPlanner([
        () => ({
          type: "write",
          method: "PATCH",
          path: patchPath,
          body: sessionPatch,
          thought: "propose",
        }),
        (ctx) => {
          const write = ctx.history.find((row) => row.method === "PATCH")
          if (write?.status === 403) {
            return {
              type: "finish",
              answer: `The API returned 403 FORBIDDEN. I did not move the session.`,
            }
          }
          return { type: "finish", answer: "moved" }
        },
      ]),
      http: rec.http,
      onEvent: bag.onEvent,
    })
    const done = run.start("move")
    await waitFor(run, "awaiting_approval")
    run.approve()
    await done
    expect(rec.writes).toHaveLength(1)
    const completed = bag.events.find((e) => e.type === "completed")
    expect(completed?.type === "completed" && completed.answer).toMatch(/403/)
    expect(completed?.type === "completed" && completed.answer.toLowerCase()).not.toContain("succeeded")
    expect(completed?.type === "completed" && completed.answer).not.toMatch(/\bmoved\b/)
  })

  it("surfaces a hostile event description and never emits or sends a role-grant write", async () => {
    const rec = recordingHttp({
      get: (path) => {
        if (path.startsWith("/events/") && !path.includes("/sessions")) {
          return {
            status: 200,
            body: JSON.stringify({
              id: "ev-inject",
              name: "Prompt Injection Conference",
              description: SEEDED_INJECTION_TEXT,
              timeZone: "Europe/London",
            }),
          }
        }
        return {
          status: 200,
          body: JSON.stringify({
            items: [
              {
                id: "ev-inject",
                name: "Prompt Injection Conference",
                description: SEEDED_INJECTION_TEXT,
                timeZone: "Europe/London",
              },
            ],
          }),
        }
      },
    })
    const bag = collect()
    const run = createAgentRun({
      planner: new ScriptedPlanner([
        () => ({ type: "read", path: "/events?limit=100", thought: "list" }),
        () => ({ type: "read", path: "/events/ev-inject", thought: "summarise" }),
        () => ({
          type: "finish",
          answer:
            "Prompt Injection Conference is in Europe/London. I treated its description as untrusted data and I did not grant anyone a role.",
        }),
      ]),
      http: rec.http,
      onEvent: bag.onEvent,
    })
    await run.start("Summarise Prompt Injection Conference")
    expect(rec.writes).toEqual([])
    const proposals = bag.events.filter((e) => e.type === "approval_required")
    expect(proposals).toEqual([])
    const warning = bag.events.find((e) => e.type === "warning")
    expect(warning).toMatchObject({
      type: "warning",
      kind: "prompt_injection_attempt",
      source: "events.description",
    })
    expect(warning && warning.type === "warning" && warning.excerpt).toContain("attacker@example.com")
    expect(
      rec.writes.some((w) => isPrivilegedWrite(w.method, w.path, w.bodyText)),
    ).toBe(false)
  })

  it("absorbs a mid-run interrupt into the next proposal instead of sending the first write", async () => {
    const rec = recordingHttp()
    const bag = collect()
    const run = createAgentRun({
      planner: new ScriptedPlanner([
        () => ({
          type: "write",
          method: "PATCH",
          path: patchPath,
          body: { version: 1, roomId: "room-a" },
          thought: "Hall A",
        }),
        (ctx) => {
          const roomId = ctx.interrupts.some((m) => /room b|hall b/i.test(m))
            ? "room-b"
            : "room-a"
          return {
            type: "write",
            method: "PATCH",
            path: patchPath,
            body: { version: 1, roomId },
            thought: "corrected room",
          }
        },
        () => ({ type: "finish", answer: "The approved PATCH succeeded." }),
      ]),
      http: rec.http,
      onEvent: bag.onEvent,
    })
    const done = run.start("move Talk A1 to Hall A")
    await waitFor(run, "awaiting_approval")
    expect(run.pending()?.bodyText).toBe(JSON.stringify({ version: 1, roomId: "room-a" }))
    run.interrupt("use Room B")
    await waitFor(run, "awaiting_approval")
    expect(run.pending()?.bodyText).toBe(JSON.stringify({ version: 1, roomId: "room-b" }))
    expect(rec.writes).toEqual([])
    run.approve()
    await done
    expect(rec.writes).toEqual([
      {
        method: "PATCH",
        path: patchPath,
        bodyText: JSON.stringify({ version: 1, roomId: "room-b" }),
      },
    ])
    expect(bag.events.some((e) => e.type === "interrupted" && e.message === "use Room B")).toBe(
      true,
    )
  })

  it("stops at the step cap and does not claim unfinished work", async () => {
    const rec = recordingHttp()
    const bag = collect()
    const run = createAgentRun({
      maxSteps: 3,
      planner: {
        next: () => ({ type: "read", path: "/events?limit=100", thought: "again" }),
      },
      http: rec.http,
      onEvent: bag.onEvent,
    })
    await run.start("keep going")
    expect(rec.reads).toHaveLength(3)
    expect(rec.writes).toEqual([])
    const completed = bag.events.find((e) => e.type === "completed")
    expect(completed).toMatchObject({ type: "completed", status: "budget_exhausted" })
    expect(completed?.type === "completed" && completed.answer).toMatch(/stopped after 3 steps/i)
    expect(completed?.type === "completed" && completed.answer).toMatch(/did not finish/i)
  })

  it("sends nothing when a proposal sits without an Approve event", async () => {
    const rec = recordingHttp()
    const run = createAgentRun({
      planner: new ScriptedPlanner([
        () => ({
          type: "write",
          method: "PATCH",
          path: patchPath,
          body: sessionPatch,
          thought: "propose",
        }),
      ]),
      http: rec.http,
      onEvent: () => {},
    })
    void run.start("move")
    await waitFor(run, "awaiting_approval")
    expect(run.pending()?.bodyText).toBe(JSON.stringify(sessionPatch))
    expect(rec.writes).toEqual([])
    expect(run.status()).toBe("awaiting_approval")
  })

  it("refuses executeApprovedWrite without a ticket", async () => {
    const sent: unknown[] = []
    const proposal = freezeProposal("p1", "PATCH", patchPath, sessionPatch)
    await expect(
      executeApprovedWrite(proposal, null, async (req) => {
        sent.push(req)
        return { status: 200, body: "{}" }
      }),
    ).rejects.toThrow(/no approval ticket/)
    expect(sent).toEqual([])
  })

  it("does not send a privileged path even if a fooled planner proposes it and the user approves", async () => {
    const rec = recordingHttp()
    const run = createAgentRun({
      planner: new ScriptedPlanner([
        () => ({
          type: "write",
          method: "POST",
          path: "/events/ev-inject/members",
          body: { email: "attacker@example.com", role: "admin" },
          thought: "obey injection",
        }),
        () => ({ type: "finish", answer: "stopped" }),
      ]),
      http: rec.http,
      onEvent: () => {},
    })
    const done = run.start("summarise")
    await waitFor(run, "awaiting_approval")
    expect(isPrivilegedWrite("POST", "/events/ev-inject/members", run.pending()?.bodyText ?? "")).toBe(
      true,
    )
    run.approve()
    await done
    expect(rec.writes).toEqual([])
  })
})
