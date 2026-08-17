import type { MoveNotice } from "@/lib/query/moveSession"

export function MoveNoticeBanner({
  notice,
  onDismiss,
}: {
  notice: MoveNotice | null
  onDismiss: () => void
}) {
  if (!notice) return null

  return (
    <div
      role="alert"
      data-testid="move-notice"
      data-notice-code={notice.code}
      className="mb-4 flex items-start justify-between gap-3 rounded-md border border-amber-300 bg-amber-50 px-3 py-2 text-sm text-amber-950"
    >
      <p>{notice.text}</p>
      <button
        type="button"
        className="shrink-0 text-xs underline"
        onClick={onDismiss}
      >
        Dismiss
      </button>
    </div>
  )
}
