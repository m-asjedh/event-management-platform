import { useState, type FormEvent } from "react"
import { useRouter } from "@tanstack/react-router"

import { Button } from "@/components/ui/button"

export function SignInForm() {
  const router = useRouter()
  const [email, setEmail] = useState("seed.admin@example.com")
  const [password, setPassword] = useState("correct-horse-battery")
  const [busy, setBusy] = useState(false)
  const [message, setMessage] = useState<string | null>(null)

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setBusy(true)
    setMessage(null)
    try {
      const res = await fetch("/api/auth/sign-in/email", {
        method: "POST",
        credentials: "include",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email, password }),
      })
      if (!res.ok) {
        setMessage("Sign-in failed. Check the email and password.")
        return
      }
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
      <label className="text-sm">
        Email
        <input
          className="mt-1 w-full rounded-md border border-neutral-300 px-2 py-1.5"
          type="email"
          name="email"
          autoComplete="username"
          value={email}
          onChange={(ev) => setEmail(ev.target.value)}
        />
      </label>
      <label className="text-sm">
        Password
        <input
          className="mt-1 w-full rounded-md border border-neutral-300 px-2 py-1.5"
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
