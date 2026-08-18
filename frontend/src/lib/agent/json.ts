export function jsonItems(body: string): Record<string, unknown>[] {
  try {
    const parsed: unknown = JSON.parse(body)
    if (!parsed || typeof parsed !== "object" || !("items" in parsed)) return []
    const items = (parsed as { items: unknown }).items
    if (!Array.isArray(items)) return []
    return items.filter(
      (row): row is Record<string, unknown> => !!row && typeof row === "object",
    )
  } catch {
    return []
  }
}

export function jsonObject(body: string): Record<string, unknown> {
  try {
    const parsed: unknown = JSON.parse(body)
    if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) return {}
    return parsed as Record<string, unknown>
  } catch {
    return {}
  }
}

export function jsonString(row: Record<string, unknown> | undefined, key: string): string {
  const value = row?.[key]
  return typeof value === "string" ? value : ""
}

export function jsonNumber(row: Record<string, unknown> | undefined, key: string): number | undefined {
  const value = row?.[key]
  return typeof value === "number" ? value : undefined
}

export function errorCode(body: string): string {
  const code = jsonString(jsonObject(body), "code")
  return code || "ERROR"
}
