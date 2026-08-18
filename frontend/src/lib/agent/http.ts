import { apiUrl } from "@/lib/api/client"
import type { WriteProposal } from "@/lib/agent/types"

export type HttpResult = { status: number; body: string }

export type AgentHttp = {
  get: (path: string, signal?: AbortSignal) => Promise<HttpResult>
  sendWrite: (req: {
    method: WriteProposal["method"]
    path: string
    bodyText: string
  }) => Promise<HttpResult>
}

export function browserHttp(): AgentHttp {
  return {
    async get(path, signal) {
      const res = await fetch(apiUrl(path), { credentials: "include", signal })
      return { status: res.status, body: await res.text() }
    },
    async sendWrite({ method, path, bodyText }) {
      const res = await fetch(apiUrl(path), {
        method,
        credentials: "include",
        headers: { "Content-Type": "application/json" },
        body: bodyText,
      })
      return { status: res.status, body: await res.text() }
    },
  }
}
