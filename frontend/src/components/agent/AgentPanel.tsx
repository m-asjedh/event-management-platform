import { useRef, useState } from "react"

import { AgentTranscript } from "@/components/agent/AgentTranscript"
import { ApprovalCard } from "@/components/agent/ApprovalCard"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import { browserHttp } from "@/lib/agent/http"
import { defaultPlanner } from "@/lib/agent/planner"
import { createAgentRun, type AgentRun } from "@/lib/agent/run"
import type { AgentEvent, AgentStatus, WriteProposal } from "@/lib/agent/types"
import { cn } from "@/lib/utils"

const examples = [
  "Which events are in America/New_York?",
  "Summarise Prompt Injection Conference",
  "Move Talk A1 to Hall B on DST Spring Forward",
]

export function AgentPanel() {
  const [question, setQuestion] = useState(examples[2] ?? "")
  const [correction, setCorrection] = useState("use Hall A")
  const [status, setStatus] = useState<AgentStatus>("idle")
  const [events, setEvents] = useState<AgentEvent[]>([])
  const [pending, setPending] = useState<WriteProposal | null>(null)
  const runRef = useRef<AgentRun | null>(null)
  const running = status === "planning" || status === "awaiting_approval" || status === "executing"

  function ask(nextQuestion: string) {
    const text = nextQuestion.trim()
    if (!text || running) return
    setEvents([])
    setPending(null)
    const run = createAgentRun({
      planner: defaultPlanner,
      http: browserHttp(),
      onStatus: setStatus,
      onEvent: (event) => {
        setEvents((prev) => [...prev, event])
        if (event.type === "approval_required") setPending(event.proposal)
        if (
          event.type === "denied" ||
          event.type === "completed" ||
          event.type === "interrupted"
        ) {
          setPending(null)
        }
      },
    })
    runRef.current = run
    void run.start(text).finally(() => {
      if (runRef.current === run) setPending(run.pending())
    })
  }

  return (
    <div>
      <form
        className="space-y-2"
        onSubmit={(event) => {
          event.preventDefault()
          ask(question)
        }}
      >
        <label className="block text-sm font-medium text-neutral-800">
          Question
          <Textarea
            className="mt-1"
            rows={3}
            value={question}
            onChange={(event) => setQuestion(event.target.value)}
          />
        </label>
        <Button type="submit" disabled={running}>
          Ask
        </Button>
      </form>
      <p className="mt-2 text-xs text-neutral-500">Try: {examples.join(" · ")}</p>

      <p
        className={cn(
          "mt-4 inline-flex rounded-full border px-2.5 py-0.5 font-mono text-xs",
          status === "awaiting_approval"
            ? "border-amber-300 bg-amber-50 text-amber-900"
            : "border-neutral-200 bg-neutral-50 text-neutral-600",
        )}
        data-testid="agent-status"
      >
        {status}
      </p>

      {pending ? (
        <ApprovalCard
          proposal={pending}
          onApprove={() => {
            runRef.current?.approve()
            setPending(null)
          }}
          onDeny={() => {
            runRef.current?.deny()
            setPending(null)
          }}
        />
      ) : null}

      <form
        className="mt-4 flex gap-2"
        onSubmit={(event) => {
          event.preventDefault()
          if (!running) return
          runRef.current?.interrupt(correction)
        }}
      >
        <Input
          className="min-w-0 flex-1"
          value={correction}
          onChange={(event) => setCorrection(event.target.value)}
          placeholder="Interrupt: use Hall A"
          aria-label="Interrupt"
        />
        <Button type="submit" variant="outline" disabled={!running}>
          Interrupt
        </Button>
      </form>

      <AgentTranscript events={events} />
    </div>
  )
}
