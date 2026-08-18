import { Link, createFileRoute } from "@tanstack/react-router"

import { AgentPanel } from "@/components/agent/AgentPanel"
import {
  NavSep,
  PageFrame,
  PageLead,
  PageNav,
  PageTitle,
  navLinkClass,
} from "@/components/layout/PageFrame"

export const Route = createFileRoute("/agent")({
  component: AgentPage,
})

function AgentPage() {
  return (
    <PageFrame width="3xl">
      <PageNav>
        <Link to="/" className={navLinkClass}>
          Events
        </Link>
        <NavSep />
        <span className="text-neutral-800">Agent</span>
      </PageNav>
      <PageTitle>Agent</PageTitle>
      <PageLead>
        The loop runs in this tab, as you, through the public API. Reads go out
        immediately. Writes pause until you approve the exact JSON. There is no
        elevated key.
      </PageLead>
      <div className="mt-6">
        <AgentPanel />
      </div>
    </PageFrame>
  )
}
