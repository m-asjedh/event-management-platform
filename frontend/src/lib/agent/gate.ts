import { isAllowedWrite } from "@/lib/agent/allowlist"
import type { ApprovalTicket, WriteProposal } from "@/lib/agent/types"

export type SendWrite = (req: {
  method: WriteProposal["method"]
  path: string
  bodyText: string
}) => Promise<{ status: number; body: string }>

/**
 * THE GATE (X6). The planner cannot call this. The only ticket issuer is
 * issueApprovalTicket, which the run loop calls after the user clicks Approve.
 * A proposal sitting in state, with no ticket, cannot reach sendWrite.
 */
export async function executeApprovedWrite(
  proposal: WriteProposal,
  ticket: ApprovalTicket | null | undefined,
  sendWrite: SendWrite,
): Promise<{ status: number; body: string }> {
  if (!ticket || ticket.granted !== true) {
    throw new Error("agent write refused: no approval ticket")
  }
  if (ticket.proposalId !== proposal.id) {
    throw new Error("agent write refused: ticket does not match proposal")
  }
  if (ticket.bodyText !== proposal.bodyText) {
    throw new Error("agent write refused: body changed after approval")
  }
  if (!isAllowedWrite(proposal.method, proposal.path)) {
    throw new Error("agent write refused: path is not an allowed API write")
  }
  return sendWrite({
    method: proposal.method,
    path: proposal.path,
    bodyText: proposal.bodyText,
  })
}

/** Issued only from the Approve control, never from planner output. */
export function issueApprovalTicket(proposal: WriteProposal): ApprovalTicket {
  return {
    granted: true,
    proposalId: proposal.id,
    bodyText: proposal.bodyText,
  }
}

export function freezeProposal(
  id: string,
  method: WriteProposal["method"],
  path: string,
  body: unknown,
): WriteProposal {
  return {
    id,
    method,
    path,
    body,
    bodyText: JSON.stringify(body),
  }
}
