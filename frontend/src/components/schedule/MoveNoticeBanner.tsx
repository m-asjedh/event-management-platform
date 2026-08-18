import { Alert } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
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
    <Alert
      data-testid="move-notice"
      data-notice-code={notice.code}
      className="mb-4 flex items-start justify-between gap-3 border-amber-300 bg-amber-50 text-amber-950"
    >
      <p>{notice.text}</p>
      <Button type="button" variant="ghost" size="sm" className="shrink-0" onClick={onDismiss}>
        Dismiss
      </Button>
    </Alert>
  )
}
