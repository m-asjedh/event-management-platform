import { Link } from "@tanstack/react-router"

import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"
import { formatDayLabel, weekContaining } from "@/lib/tz/eventZone"

type DayNavProps = {
  eventId: string
  day: string
}

export function DayNav({ eventId, day }: DayNavProps) {
  const days = weekContaining(day)

  return (
    <nav aria-label="Days of the week" className="flex flex-wrap gap-1">
      {days.map((ymd) => {
        const selected = ymd === day
        return (
          <Button
            key={ymd}
            variant={selected ? "default" : "outline"}
            size="default"
            className={cn("h-8 px-2.5 text-xs", selected && "pointer-events-none")}
            asChild
          >
            <Link
              to="/events/$eventId/schedule"
              params={{ eventId }}
              search={{ day: ymd }}
              aria-current={selected ? "date" : undefined}
            >
              {formatDayLabel(ymd)}
            </Link>
          </Button>
        )
      })}
    </nav>
  )
}
