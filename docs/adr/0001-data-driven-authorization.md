# ADR 0001 — Centralized Data-Driven Authorization

## Decision

Authorization is one chokepoint. A handler asks `Grant.Can(permission)` for `user + event + action`. It never switches on a role name.

Roles and permissions are rows:

- `roles`
- `permissions`
- `role_permissions`
- `event_members` (role per event; there is no global role column)

The same `Can()` check decides the response body. `PresentEvent` drops the roster unless `member.read` is granted, and clears emails unless `user.email.read` is granted.

Adding a fourth role is an `INSERT` into `roles` / `role_permissions`. Handlers do not change.

## Rejected Alternatives

**Role checks inside handlers**

```go
if role == "admin" {
    ...
}
```

This spreads permission logic across every endpoint. Adding a role would mean editing handlers. `TestGrantDoesNotSwitchOnRoleName` is the opposite: a new role with the same permission rows behaves the same with no code change.

**Hard-coded Go permission map**

A central map is better than handler-level `if`s, but the rules would still live in the binary. Changing who may see emails would require a deploy.

**OPA / Casbin**

Too much machinery for three roles and a grant table.

## Why

Roles belong to an event, not to a user. The same person can be admin on Event A and attendee on Event B.

`TestAttendeeEventOmitsRosterAndEmails` asserts on the body, not on which URL returned 200.
