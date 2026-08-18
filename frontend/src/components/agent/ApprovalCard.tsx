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
      className="mt-4 rounded-md border border-amber-300 bg-amber-50 p-4"
    >
      <h2 className="font-semibold">Approval needed</h2>
      <p className="mt-1 font-mono text-sm">
        {proposal.method} {proposal.path}
      </p>
      <pre className="mt-3 overflow-auto rounded bg-white p-3 text-xs">
        {proposal.bodyText}
      </pre>
      <p className="mt-2 text-xs text-neutral-600">
        This JSON is what will be sent. Approving does not regenerate it.
      </p>
      <div className="mt-3 flex gap-2">
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
