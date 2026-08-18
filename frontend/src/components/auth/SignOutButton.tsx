import { useQueryClient } from "@tanstack/react-query"
import { useRouter } from "@tanstack/react-router"
import { useState } from "react"

import { Button } from "@/components/ui/button"
import { clearClientSession, postSignOut } from "@/lib/auth/session"

export function SignOutButton() {
  const router = useRouter()
  const queryClient = useQueryClient()
  const [busy, setBusy] = useState(false)

  async function onClick() {
    if (busy) return
    setBusy(true)
    try {
      await postSignOut()
      await clearClientSession(queryClient, null)
      await router.navigate({ to: "/" })
      await router.invalidate()
    } finally {
      setBusy(false)
    }
  }

  return (
    <Button type="button" variant="ghost" size="sm" onClick={onClick} disabled={busy}>
      Sign out
    </Button>
  )
}
