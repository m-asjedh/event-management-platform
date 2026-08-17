import { isApiError } from "@/lib/api/error"
import { SignInForm } from "@/components/auth/SignInForm"

export function ApiErrorView({ error }: { error: Error }) {
  if (!isApiError(error)) {
    return (
      <main className="mx-auto max-w-xl p-8">
        <h1 className="text-2xl font-semibold">Something went wrong</h1>
        <p className="mt-2 font-mono text-sm text-neutral-700">{error.message}</p>
      </main>
    )
  }

  return (
    <main className="mx-auto max-w-xl p-8">
      <h1 className="text-2xl font-semibold">{error.body.code}</h1>
      <p className="mt-2 text-neutral-700">{error.body.reason}</p>
      <p className="mt-1 font-mono text-sm text-neutral-500">{error.body.code}</p>
      {error.body.code === "UNAUTHENTICATED" ? (
        <div className="mt-6">
          <SignInForm />
        </div>
      ) : null}
    </main>
  )
}
