import { Badge } from "@/components/ui/badge"
import type { Invitation } from "@/lib/api/types"

function statusVariant(status: Invitation["status"]) {
  switch (status) {
    case "accepted":
      return "success" as const
    case "pending":
      return "warning" as const
    case "declined":
    case "revoked":
      return "destructive" as const
  }
}

export function InvitationRow({ invitation }: { invitation: Invitation }) {
  return (
    <div
      data-invitation-id={invitation.id}
      data-testid="invitation-row"
      className="flex h-12 items-center gap-4 border-b border-neutral-100 px-3 text-sm last:border-b-0"
    >
      <span className="min-w-0 flex-1 truncate font-mono text-xs text-neutral-800">
        {invitation.email ?? "Email hidden"}
      </span>
      <span className="w-28 shrink-0 text-neutral-600">{invitation.role}</span>
      <span className="w-24 shrink-0">
        <Badge variant={statusVariant(invitation.status)} className="normal-case tracking-normal">
          {invitation.status}
        </Badge>
      </span>
    </div>
  )
}
