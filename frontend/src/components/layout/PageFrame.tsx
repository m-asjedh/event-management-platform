import type { ReactNode } from "react"

import { cn } from "@/lib/utils"

export function PageFrame({
  children,
  width = "xl",
}: {
  children: ReactNode
  width?: "xl" | "3xl" | "4xl" | "6xl"
}) {
  const max =
    width === "6xl"
      ? "max-w-6xl"
      : width === "4xl"
        ? "max-w-4xl"
        : width === "3xl"
          ? "max-w-3xl"
          : "max-w-xl"
  return <main className={cn("mx-auto w-full px-6 py-8", max)}>{children}</main>
}

export function PageNav({ children }: { children: ReactNode }) {
  return (
    <p className="flex flex-wrap items-center gap-x-1 text-sm text-neutral-500">
      {children}
    </p>
  )
}

export function NavSep() {
  return <span aria-hidden="true">·</span>
}

export function PageTitle({ children }: { children: ReactNode }) {
  return <h1 className="mt-3 text-2xl font-semibold tracking-tight">{children}</h1>
}

export function PageLead({ children }: { children: ReactNode }) {
  return <p className="mt-1 text-sm leading-6 text-neutral-600">{children}</p>
}

export const navLinkClass =
  "rounded-sm underline-offset-4 hover:text-neutral-900 hover:underline"
