import { ApiError } from "@/lib/api/error"

const base = import.meta.env.VITE_API_BASE ?? ""

export function apiUrl(path: string): string {
  return `${base}${path}`
}

export async function getJSON<T>(path: string): Promise<T> {
  const res = await fetch(apiUrl(path), { credentials: "include" })
  if (!res.ok) {
    throw await ApiError.fromResponse(path, res)
  }
  return res.json() as Promise<T>
}

export async function patchJSON<T>(path: string, body: unknown): Promise<T> {
  const res = await fetch(apiUrl(path), {
    method: "PATCH",
    credentials: "include",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  })
  if (!res.ok) {
    throw await ApiError.fromResponse(path, res)
  }
  return res.json() as Promise<T>
}
