import type { AgentEvent } from "@/lib/agent/types"

export function AgentTranscript({ events }: { events: AgentEvent[] }) {
  return (
    <ol className="mt-4 space-y-2 text-sm" data-testid="agent-transcript">
      {events.map((event, index) => (
        <li key={`${event.type}-${index}`} className="rounded-md border px-3 py-2">
          <TranscriptRow event={event} />
        </li>
      ))}
    </ol>
  )
}

function TranscriptRow({ event }: { event: AgentEvent }) {
  switch (event.type) {
    case "thought":
      return <p className="text-neutral-700">{event.text}</p>
    case "tool_started":
      return (
        <p className="font-mono text-xs">
          → {event.method} {event.path}
        </p>
      )
    case "tool_completed":
      return (
        <p className="font-mono text-xs">
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
