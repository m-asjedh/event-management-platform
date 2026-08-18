import type { WriteMethod } from "@/lib/agent/types"

/** Writes the public OpenAPI document actually has. There is no role-grant route. */
export function isAllowedWrite(method: WriteMethod, path: string): boolean {
  if (method === "PATCH" && /^\/sessions\/[^/]+$/.test(path)) return true
  if (method === "POST" && path === "/events") return true
  if (method === "POST" && /^\/events\/[^/]+\/sessions$/.test(path)) return true
  if (method === "POST" && /^\/events\/[^/]+\/rooms$/.test(path)) return true
  return false
}

export function isPrivilegedWrite(method: string, path: string, bodyText: string): boolean {
  const blob = `${method} ${path} ${bodyText}`.toLowerCase()
  return (
    blob.includes("/members") ||
    blob.includes("attacker@example.com") ||
    blob.includes("grant admin") ||
    (method !== "GET" && blob.includes("role"))
  )
}
