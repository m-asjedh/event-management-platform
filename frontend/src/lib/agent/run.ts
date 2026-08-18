import { executeApprovedWrite, freezeProposal, issueApprovalTicket } from "@/lib/agent/gate"
import type { AgentHttp } from "@/lib/agent/http"
import { findInjection } from "@/lib/agent/injection"
import { errorCode } from "@/lib/agent/json"
import {
  AGENT_MAX_STEPS,
  type AgentAction,
  type AgentEvent,
  type AgentObservation,
  type AgentStatus,
  type Planner,
  type WriteProposal,
} from "@/lib/agent/types"

export type AgentRun = {
  start: (question: string) => Promise<void>
  approve: () => void
  deny: () => void
  interrupt: (message: string) => void
  status: () => AgentStatus
  pending: () => WriteProposal | null
}

type Decision = "approve" | "deny" | "interrupt"

export function createAgentRun(opts: {
  planner: Planner
  http: AgentHttp
  maxSteps?: number
  onEvent: (event: AgentEvent) => void
  onStatus?: (status: AgentStatus) => void
}): AgentRun {
  const maxSteps = opts.maxSteps ?? AGENT_MAX_STEPS
  let status: AgentStatus = "idle"
  let pending: WriteProposal | null = null
  let proposalSeq = 0
  const interrupts: string[] = []
  let waiter: ((decision: Decision) => void) | null = null

  function setStatus(next: AgentStatus) {
    status = next
    opts.onStatus?.(next)
  }

  function emit(event: AgentEvent) {
    opts.onEvent(event)
  }

  function approve() {
    if (status !== "awaiting_approval" || !pending) return
    waiter?.("approve")
  }

  function deny() {
    if (status !== "awaiting_approval" || !pending) return
    waiter?.("deny")
  }

  function interrupt(message: string) {
    const text = message.trim()
    if (!text) return
    interrupts.push(text)
    emit({ type: "interrupted", message: text })
    if (status === "awaiting_approval" && waiter) {
      pending = null
      setStatus("planning")
      waiter("interrupt")
    }
  }

  function waitForDecision(): Promise<Decision> {
    return new Promise((resolve) => {
      waiter = (decision) => {
        waiter = null
        resolve(decision)
      }
    })
  }

  async function dispatchRead(path: string): Promise<AgentObservation> {
    emit({ type: "tool_started", method: "GET", path })
    const result = await opts.http.get(path)
    const obs: AgentObservation = {
      method: "GET",
      path,
      status: result.status,
      body: result.body,
    }
    emit({
      type: "tool_completed",
      method: "GET",
      path,
      status: result.status,
      summary: summarise(obs),
    })
    const hit = findInjection(result.body)
    if (hit) {
      emit({
        type: "warning",
        kind: "prompt_injection_attempt",
        source: hit.source,
        excerpt: hit.excerpt,
      })
    }
    return obs
  }

  async function start(question: string) {
    const history: AgentObservation[] = []
    setStatus("planning")

    for (let step = 0; step < maxSteps; step++) {
      setStatus("planning")
      const action: AgentAction = opts.planner.next({
        question,
        history,
        interrupts: [...interrupts],
      })
      if (action.type === "finish") {
        emit({ type: "completed", status: "done", answer: action.answer })
        setStatus("done")
        return
      }
      emit({ type: "thought", text: action.thought })

      if (action.type === "read") {
        history.push(await dispatchRead(action.path))
        continue
      }

      const proposal = freezeProposal(
        `proposal-${++proposalSeq}`,
        action.method,
        action.path,
        action.body,
      )
      pending = proposal
      setStatus("awaiting_approval")
      emit({ type: "approval_required", proposal })
      const decision = await waitForDecision()
      const held = proposal
      pending = null

      if (decision === "deny") {
        emit({ type: "denied", proposal: held })
        history.push({
          method: held.method,
          path: held.path,
          status: 0,
          body: "",
          denied: true,
        })
        continue
      }

      if (decision === "interrupt") {
        history.push({
          method: held.method,
          path: held.path,
          status: 0,
          body: "",
          superseded: true,
        })
        continue
      }

      // Gate: issueApprovalTicket is only reachable after Approve.
      setStatus("executing")
      emit({ type: "tool_started", method: held.method, path: held.path })
      const ticket = issueApprovalTicket(held)
      let result: { status: number; body: string }
      try {
        result = await executeApprovedWrite(held, ticket, opts.http.sendWrite)
      } catch (err) {
        result = {
          status: 0,
          body: JSON.stringify({
            code: "INTERNAL",
            reason: err instanceof Error ? err.message : "write refused",
          }),
        }
      }
      const obs: AgentObservation = {
        method: held.method,
        path: held.path,
        status: result.status,
        body: result.body,
      }
      history.push(obs)
      emit({
        type: "tool_completed",
        method: held.method,
        path: held.path,
        status: result.status,
        summary: summarise(obs),
      })
    }

    const answer = `I stopped after ${maxSteps} steps. I will not summarise work I did not finish.`
    emit({ type: "completed", status: "budget_exhausted", answer })
    setStatus("budget_exhausted")
  }

  return {
    start,
    approve,
    deny,
    interrupt,
    status: () => status,
    pending: () => pending,
  }
}

function summarise(obs: AgentObservation): string {
  if (obs.status === 403) return "403 FORBIDDEN"
  if (obs.status === 401) return "401 UNAUTHENTICATED"
  if (obs.status === 404) return "404 NOT_FOUND"
  if (obs.status === 409) return `409 ${errorCode(obs.body)}`
  if (obs.status >= 400) return `${obs.status} ${errorCode(obs.body)}`
  return String(obs.status)
}
