/**
 * Wall-clock in an event's IANA zone. Instant is RFC 3339; the zone comes from
 * GET /events/{id}. Never use Date#getHours / toLocaleString without timeZone —
 * those follow the machine, not the event.
 */

export type ZonedParts = {
  year: number
  month: number
  day: number
  hour: number
  minute: number
}

function pad2(n: number): string {
  return n.toString().padStart(2, "0")
}

export function zonedParts(instant: string | Date, timeZone: string): ZonedParts {
  const date = typeof instant === "string" ? new Date(instant) : instant
  const fmt = new Intl.DateTimeFormat("en-GB", {
    timeZone,
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    hourCycle: "h23",
  })
  const bag: Record<string, string> = {}
  for (const part of fmt.formatToParts(date)) {
    if (part.type !== "literal") {
      bag[part.type] = part.value
    }
  }
  return {
    year: Number(bag.year),
    month: Number(bag.month),
    day: Number(bag.day),
    hour: Number(bag.hour),
    minute: Number(bag.minute),
  }
}

export function instantToYmd(instant: string | Date, timeZone: string): string {
  const p = zonedParts(instant, timeZone)
  return `${p.year}-${pad2(p.month)}-${pad2(p.day)}`
}

export function formatEventTime(instant: string | Date, timeZone: string): string {
  const p = zonedParts(instant, timeZone)
  return `${pad2(p.hour)}:${pad2(p.minute)}`
}

export function minutesFromMidnight(
  instant: string | Date,
  timeZone: string,
): number {
  const p = zonedParts(instant, timeZone)
  return p.hour * 60 + p.minute
}

export function sameEventDay(
  instant: string | Date,
  timeZone: string,
  ymd: string,
): boolean {
  return instantToYmd(instant, timeZone) === ymd
}

/** Monday–Sunday civil week containing ymd (YYYY-MM-DD). Weekday of a civil date is zone-independent. */
export function weekContaining(ymd: string): string[] {
  const [y, m, d] = ymd.split("-").map(Number)
  const utc = new Date(Date.UTC(y, m - 1, d))
  const dow = utc.getUTCDay()
  const mondayOffset = dow === 0 ? -6 : 1 - dow
  const days: string[] = []
  for (let i = 0; i < 7; i++) {
    const day = new Date(utc)
    day.setUTCDate(utc.getUTCDate() + mondayOffset + i)
    days.push(day.toISOString().slice(0, 10))
  }
  return days
}

export function formatDayLabel(ymd: string): string {
  const [y, m, d] = ymd.split("-").map(Number)
  const utc = new Date(Date.UTC(y, m - 1, d, 12))
  return new Intl.DateTimeFormat("en-GB", {
    weekday: "short",
    day: "numeric",
    month: "short",
    timeZone: "UTC",
  }).format(utc)
}
