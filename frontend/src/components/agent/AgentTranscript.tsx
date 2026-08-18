import type { AgentEvent } from "@/lib/agent/types"
import { cn } from "@/lib/utils"

export function AgentTranscript({ events }: { events: AgentEvent[] }) {
  return (
    <ol className="mt-4 space-y-2 text-sm" data-testid="agent-transcript">
      {events.map((event, index) => (
        <li
          key={`${event.type}-${index}`}
          className={cn("rounded-lg border px-3 py-2", toneClass(event.type))}
        >
          <p className="mb-1 text-[10px] font-semibold tracking-wide text-neutral-500 uppercase">
            {labelFor(event.type)}
          </p>
          <TranscriptRow event={event} />
        </li>
      ))}
    </ol>
  )
}

function labelFor(type: AgentEvent["type"]): string {
  switch (type) {
    case "thought":
      return "Thought"
    case "tool_started":
      return "Call"
    case "tool_completed":
      return "Result"
    case "approval_required":
      return "Approval"
    case "denied":
      return "Denied"
    case "interrupted":
      return "Interrupt"
    case "warning":
      return "Warning"
    case "completed":
      return "Answer"
  }
}

function toneClass(type: AgentEvent["type"]): string {
  switch (type) {
    case "thought":
      return "border-neutral-200 bg-neutral-50"
    case "tool_started":
      return "border-sky-200 bg-sky-50"
    case "tool_completed":
      return "border-emerald-200 bg-emerald-50"
    case "approval_required":
      return "border-amber-300 bg-amber-50"
    case "denied":
      return "border-red-200 bg-red-50"
    case "interrupted":
      return "border-amber-200 bg-amber-50"
    case "warning":
      return "border-amber-300 bg-amber-50"
    case "completed":
      return "border-neutral-300 bg-white shadow-sm"
  }
}

function TranscriptRow({ event }: { event: AgentEvent }) {
  switch (event.type) {
    case "thought":
      return <p className="text-neutral-700 italic">{event.text}</p>
    case "tool_started":
      return (
        <p className="font-mono text-xs text-sky-950">
          → {event.method} {event.path}
        </p>
      )
    case "tool_completed":
      return (
        <p className="font-mono text-xs text-emerald-950">
          ← {event.method} {event.path} · {event.summary}
        </p>
      )
    case "approval_required":
      return (
        <p>
          Waiting for approval of {event.proposal.method} {event.proposal.path}
        </p>
      )
    case "denied":
      return (
        <p>
          Denied {event.proposal.method} {event.proposal.path}. It was not sent.
        </p>
      )
    case "interrupted":
      return <p>Interrupt absorbed: {event.message}</p>
    case "warning":
      return (
        <details>
          <summary className="cursor-pointer text-amber-800">
            An event description tried to instruct the assistant. It was not granted anything.
          </summary>
          <pre className="mt-2 overflow-auto text-xs text-neutral-700">{event.excerpt}</pre>
        </details>
      )
    case "completed":
      return (
        <p>
          <span className="font-medium">Answer: </span>
          {event.answer}
        </p>
      )
  }
}
