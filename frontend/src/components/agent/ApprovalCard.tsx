import { Button } from "@/components/ui/button"
import type { WriteProposal } from "@/lib/agent/types"

export function ApprovalCard({
  proposal,
  onApprove,
  onDeny,
}: {
  proposal: WriteProposal
  onApprove: () => void
  onDeny: () => void
}) {
  return (
    <section
      data-testid="approval-card"
      className="mt-4 rounded-xl border-2 border-amber-500 bg-amber-50 p-5 shadow-[0_0_0_4px_rgba(245,158,11,0.18)]"
    >
      <p className="text-[11px] font-semibold tracking-wide text-amber-800 uppercase">
        Write paused
      </p>
      <h2 className="mt-1 text-lg font-semibold text-amber-950">Approval needed</h2>
      <p className="mt-2 font-mono text-sm text-amber-950">
        {proposal.method} {proposal.path}
      </p>
      <pre className="mt-3 overflow-auto rounded-md border border-amber-200 bg-white p-3 font-mono text-xs leading-5 text-neutral-900">
        {proposal.bodyText}
      </pre>
      <p className="mt-2 text-xs text-neutral-600">
        This JSON is what will be sent. Approving does not regenerate it.
      </p>
      <div className="mt-4 flex flex-wrap gap-2">
        <Button type="button" onClick={onApprove}>
          Approve
        </Button>
        <Button type="button" variant="outline" onClick={onDeny}>
          Deny
        </Button>
      </div>
    </section>
  )
}
