# AI Workflow

Written as the work happens, not reconstructed at the end.

---

## Tools


| Tool   | Model    | Used for                                                                    |
| ------ | -------- | --------------------------------------------------------------------------- |
| Claude | Opus 5   | Reading the requirements, planning, reviewing decisions before writing code |
| Cursor | Grok 4.6 | Writing code                                                                |
| git-ai | 1.6.22   | Line-level AI/human attribution, in `refs/notes/ai` from the first commit   |


```bash
git log --show-notes=ai   # attribution per commit
```

---

## What I drove vs. what I delegated

*As the work happens.*

---

## How I planned

*As the work happens.*

---

## Where the AI produced wrong code

Rules for this section: paste the actual code, not a description of it. Say why it looked correct — if
it looked obviously wrong it is not worth recording. Say how I noticed. Name the test that catches it
now.

*Nothing yet.*

---

## Log


| Date       | Notes                                                                              |
| ---------- | ---------------------------------------------------------------------------------- |
| 2026-08-17 | git-ai installed and verified before the first commit. Repo created, notes pushed. |
| 2026-08-17 | Seed via COPY (~800ms, 50k invitations). Shared scrypt hash is a seed-speed choice. uuidv7 minted in Go so COPY has parent IDs and reruns match. Invitations keyset EXPLAIN captured in docs/postgres-18.md. |
