import { Link, createFileRoute } from "@tanstack/react-router"

import { AgentPanel } from "@/components/agent/AgentPanel"

export const Route = createFileRoute("/agent")({
  component: AgentPage,
})

function AgentPage() {
  return (
    <main className="mx-auto max-w-3xl p-6">
      <p className="text-sm text-neutral-500">
        <Link to="/" className="underline">
          Events
        </Link>
      </p>
      <h1 className="mt-2 text-2xl font-semibold">Agent</h1>
      <p className="mt-1 text-sm text-neutral-600">
        The loop runs in this tab, as you, through the public API. Reads go out
        immediately. Writes pause until you approve the exact JSON. There is no
        elevated key.
      </p>
      <div className="mt-6">
        <AgentPanel />
      </div>
    </main>
  )
}
