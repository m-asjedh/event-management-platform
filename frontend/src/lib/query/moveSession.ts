import { useQueryClient, useMutation } from "@tanstack/react-query"
import { useCallback, useState } from "react"

import { patchJSON } from "@/lib/api/client"
import { isApiError } from "@/lib/api/error"
import type {
  ErrorBody,
  PatchedSession,
  Room,
  Session,
  SessionList,
  SessionPatch,
} from "@/lib/api/types"
import { queryKeys } from "@/lib/query/keys"
import type { MoveSessionInput } from "@/lib/schedule/drop"
import { wallClockToInstant } from "@/lib/tz/eventZone"

export type MoveNotice = {
  code: ErrorBody["code"]
  text: string
}

type MoveContext = {
  previous: SessionList | undefined
}

function roomName(rooms: Room[], roomId: string | null): string {
  if (!roomId) return "Unplaced"
  return rooms.find((room) => room.id === roomId)?.name ?? "that room"
}

function wallTime(wall: string): string {
  const match = /T(\d{2}:\d{2})/.exec(wall)
  return match?.[1] ?? wall
}

function replaceSession(list: SessionList | undefined, next: Session): SessionList {
  const items = list?.items ?? []
  const index = items.findIndex((session) => session.id === next.id)
  if (index === -1) return { items: [...items, next] }
  const copy = items.slice()
  copy[index] = next
  return { items: copy }
}

function noticeForError(
  error: unknown,
  input: MoveSessionInput,
  rooms: Room[],
): MoveNotice {
  if (!isApiError(error)) {
    return { code: "INTERNAL", text: error instanceof Error ? error.message : "Move failed" }
  }
  const { code, reason, conflict, fieldErrors } = error.body
  switch (code) {
    case "ROOM_CONFLICT": {
      const where = `${roomName(rooms, input.roomId)} at ${wallTime(input.startLocal)}`
      const taken = conflict?.conflictingTitle
        ? `${where} is taken (${conflict.conflictingTitle})`
        : `${where} is taken`
      return { code, text: `${taken}. Your session is back in its original slot.` }
    }
    case "STALE_VERSION":
      return {
        code,
        text: "The schedule changed — here's the current state. Your edit was based on an old version.",
      }
    case "FORBIDDEN":
      return { code, text: "You can't move this session." }
    case "VALIDATION_ERROR": {
      const fields = (fieldErrors ?? [])
        .map((item) => `${item.field}: ${item.reason}`)
        .join("; ")
      return { code, text: fields || reason }
    }
    default:
      return { code, text: reason }
  }
}

export function useMoveSession(opts: {
  eventId: string
  timeZone: string
  rooms: Room[]
}) {
  const queryClient = useQueryClient()
  const [notice, setNotice] = useState<MoveNotice | null>(null)
  const clearNotice = useCallback(() => setNotice(null), [])
  const key = queryKeys.sessions(opts.eventId)

  const mutation = useMutation<PatchedSession, Error, MoveSessionInput, MoveContext>({
    retry: false,
    mutationFn: (input) => {
      const body: SessionPatch = {
        version: input.session.version,
        roomId: input.roomId,
        startsAt: input.startLocal,
        endsAt: input.endLocal,
      }
      return patchJSON<PatchedSession>(`/sessions/${input.session.id}`, body)
    },
    onMutate: async (input) => {
      setNotice(null)
      await queryClient.cancelQueries({ queryKey: key })
      const previous = queryClient.getQueryData<SessionList>(key)
      const optimistic: Session = {
        ...input.session,
        roomId: input.roomId,
        startsAt: wallClockToInstant(opts.timeZone, input.startLocal),
        endsAt: wallClockToInstant(opts.timeZone, input.endLocal),
      }
      queryClient.setQueryData<SessionList>(key, replaceSession(previous, optimistic))
      return { previous }
    },
    onError: (error, input, context) => {
      if (isApiError(error) && error.body.code === "STALE_VERSION") {
        const fresh = error.body.conflict?.currentState
        if (fresh) {
          queryClient.setQueryData<SessionList>(key, (current) =>
            replaceSession(context?.previous ?? current, fresh),
          )
        } else if (context?.previous) {
          queryClient.setQueryData(key, context.previous)
        }
      } else if (context?.previous) {
        queryClient.setQueryData(key, context.previous)
      }
      setNotice(noticeForError(error, input, opts.rooms))
    },
    onSuccess: (server) => {
      queryClient.setQueryData<SessionList>(key, (current) => replaceSession(current, server))
    },
  })

  return { ...mutation, notice, clearNotice }
}
