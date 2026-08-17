# ADR 0003 — OpenAPI as the API Source of Truth

Supporting record. The required three for this submission are 0001, 0002, and 0006.

## Decision

A single OpenAPI document is the source of truth for the HTTP API, including the error envelope from ADR 0002.

From that file we generate:

- Go request/response models (and the server interface, if the generator earns its keep)
- TypeScript frontend types

CI must send real requests to the running server and check responses against the same file. A spec that is never executed does not meet the assignment's wording: a contract that CI verifies the server actually honours.

Sign-up and sign-in stay with Better Auth. They are not part of this document.

## Rejected Alternatives

**Code-first, generate the spec afterwards**

Easy to start. The frontend then tracks whatever the last handler happened to return, including accidental fields. Drift is the default.

**Handwritten TypeScript interfaces**

Handwritten response interfaces defeat the point of the contract.

## Why

Three copies of the same API — Go structs, TS types, markdown — will disagree. One file, generated types, and a CI check against the live server is the combination that can actually fail a pull request.
