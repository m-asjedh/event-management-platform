import type { Invitation } from "@/lib/api/types"

export function InvitationRow({ invitation }: { invitation: Invitation }) {
  return (
    <div
      data-invitation-id={invitation.id}
      data-testid="invitation-row"
      className="flex h-12 items-center gap-4 border-b px-3 text-sm"
    >
      <span className="min-w-0 flex-1 truncate font-mono text-xs">
        {invitation.email ?? "Email hidden"}
      </span>
      <span className="w-28 shrink-0 text-neutral-600">{invitation.role}</span>
      <span className="w-24 shrink-0 text-neutral-600">{invitation.status}</span>
    </div>
  )
}
