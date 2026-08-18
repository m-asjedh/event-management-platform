import { isApiError } from "@/lib/api/error"
import { SignInForm } from "@/components/auth/SignInForm"
import { Alert } from "@/components/ui/alert"

export function ApiErrorView({ error }: { error: Error }) {
  if (!isApiError(error)) {
    return (
      <main className="mx-auto max-w-xl px-6 py-8">
        <Alert>
          <h1 className="text-lg font-semibold">Something went wrong</h1>
          <p className="mt-2 font-mono text-sm text-neutral-700">{error.message}</p>
        </Alert>
      </main>
    )
  }

  return (
    <main className="mx-auto max-w-xl px-6 py-8">
      <Alert className="border-red-200 bg-red-50">
        <h1 className="text-lg font-semibold text-red-950">{error.body.code}</h1>
        <p className="mt-2 text-neutral-700">{error.body.reason}</p>
        <p className="mt-1 font-mono text-sm text-neutral-500">{error.body.code}</p>
      </Alert>
      {error.body.code === "UNAUTHENTICATED" ? (
        <div className="mt-6 rounded-xl border border-neutral-200 bg-white p-5 shadow-sm">
          <SignInForm />
        </div>
      ) : null}
    </main>
  )
}
