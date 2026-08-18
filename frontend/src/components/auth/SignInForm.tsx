import { useQueryClient } from "@tanstack/react-query"
import { useRouter } from "@tanstack/react-router"
import { useState, type FormEvent } from "react"

import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { clearClientSession, postSignIn, postSignOut } from "@/lib/auth/session"
import { meQuery } from "@/lib/query/me"

export function SignInForm() {
  const router = useRouter()
  const queryClient = useQueryClient()
  const [email, setEmail] = useState("seed.admin@example.com")
  const [password, setPassword] = useState("correct-horse-battery")
  const [busy, setBusy] = useState(false)
  const [message, setMessage] = useState<string | null>(null)

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setBusy(true)
    setMessage(null)
    try {
      // Sign out first so an existing session cannot mix with the new login.
      await postSignOut()
      const res = await postSignIn(email, password)
      if (!res.ok) {
        await clearClientSession(queryClient, null)
        setMessage("Sign-in failed. Check the email and password.")
        await router.navigate({ to: "/" })
        await router.invalidate()
        return
      }
      await clearClientSession(queryClient)
      await queryClient.fetchQuery(meQuery)
      await router.navigate({ to: "/" })
      await router.invalidate()
    } finally {
      setBusy(false)
    }
  }

  return (
    <form onSubmit={onSubmit} className="flex max-w-sm flex-col gap-3">
      <p className="text-sm text-neutral-600">
        Better Auth cookie via the Vite proxy. Demo password is in the README.
      </p>
      <label className="text-sm font-medium text-neutral-800">
        Email
        <Input
          className="mt-1"
          type="email"
          name="email"
          autoComplete="username"
          value={email}
          onChange={(ev) => setEmail(ev.target.value)}
        />
      </label>
      <label className="text-sm font-medium text-neutral-800">
        Password
        <Input
          className="mt-1"
          type="password"
          name="password"
          autoComplete="current-password"
          value={password}
          onChange={(ev) => setPassword(ev.target.value)}
        />
      </label>
      {message ? <p className="text-sm text-red-700">{message}</p> : null}
      <Button type="submit" disabled={busy}>
        {busy ? "Signing in…" : "Sign in"}
      </Button>
    </form>
  )
}
