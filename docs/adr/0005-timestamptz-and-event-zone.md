# ADR 0005 — Instants and Event Time Zones

## Decision

Event and session times are stored as `timestamptz`.

Each event also stores its IANA zone (`Europe/London`, `America/New_York`, `Asia/Colombo`, …) in `events.time_zone`.

A CHECK cannot query `pg_timezone_names`, so validity is a foreign key onto `time_zones`, filled from that catalog.

Local wall-clock time is not stored. It is `(timestamptz, event.time_zone)` at the edge.

## Rejected Alternatives

**Naive local timestamps**

This drops the zone. In DST a local time can fail to exist, or exist twice, and the row has no way to say which instant was meant.

**UTC only, no zone column**

The instant is correct. "9am in the event's city" has nothing to resolve against.

## Why

Events belong to a place, not to UTC. Storing both the instant and the event's zone is what display and DST handling need.
