import type { ErrorBody } from "@/lib/api/types"

const codes: ReadonlySet<ErrorBody["code"]> = new Set([
  "UNAUTHENTICATED",
  "FORBIDDEN",
  "NOT_FOUND",
  "INTERNAL",
  "VALIDATION_ERROR",
  "ROOM_CONFLICT",
  "STALE_VERSION",
])

function asCode(value: unknown): ErrorBody["code"] {
  if (typeof value === "string" && codes.has(value as ErrorBody["code"])) {
    return value as ErrorBody["code"]
  }
  return "INTERNAL"
}

export class ApiError extends Error {
  readonly status: number
  readonly body: ErrorBody

  constructor(status: number, body: ErrorBody) {
    super(body.reason)
    this.name = "ApiError"
    this.status = status
    this.body = body
  }

  static async fromResponse(path: string, res: Response): Promise<ApiError> {
    let parsed: unknown
    try {
      parsed = await res.json()
    } catch {
      return new ApiError(res.status, {
        code: "INTERNAL",
        reason: `${path} ${res.status}`,
      })
    }
    if (!parsed || typeof parsed !== "object") {
      return new ApiError(res.status, {
        code: "INTERNAL",
        reason: `${path} ${res.status}`,
      })
    }
    const rec = parsed as Record<string, unknown>
    const reason =
      typeof rec.reason === "string" && rec.reason.length > 0
        ? rec.reason
        : `${path} ${res.status}`
    const body: ErrorBody = { code: asCode(rec.code), reason }
    if (Array.isArray(rec.fieldErrors)) {
      body.fieldErrors = rec.fieldErrors as ErrorBody["fieldErrors"]
    }
    if (rec.conflict && typeof rec.conflict === "object") {
      body.conflict = rec.conflict as ErrorBody["conflict"]
    }
    return new ApiError(res.status, body)
  }
}

export function isApiError(error: unknown): error is ApiError {
  return error instanceof ApiError
}
