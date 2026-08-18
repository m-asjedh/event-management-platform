import type { InputHTMLAttributes } from "react"

import { cn } from "@/lib/utils"

export function Input({ className, ...props }: InputHTMLAttributes<HTMLInputElement>) {
  return (
    <input
      className={cn(
        "h-9 w-full rounded-md border border-neutral-200 bg-white px-3 text-sm shadow-sm outline-none",
        "placeholder:text-neutral-400",
        "focus-visible:border-neutral-400 focus-visible:ring-2 focus-visible:ring-neutral-900/10",
        "disabled:cursor-not-allowed disabled:opacity-50",
        className,
      )}
      {...props}
    />
  )
}
