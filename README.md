# Event Management Platform

Events — conferences, weddings, gatherings — with sessions scheduled inside them, rooms, and people.

Roles are per event, not global: the same user can be an admin on one event and an attendee on another.

## Stack

Go · PostgreSQL 18 · OpenAPI · React + Vite · TypeScript

## Running it

```bash
make up && make seed && make test
```

Requires Docker, Docker Compose and make. Nothing else.

`make seed` only talks to the local Compose database. It truncates first so it can rerun. All seed accounts share one real scrypt hash to keep seeding fast; not a production pattern. Demo password: `correct-horse-battery`.

_Under construction._
