import { describe, expect, it } from "vitest"

import { validateInvitationsSearch } from "@/lib/invitations/search"

describe("validateInvitationsSearch", () => {
  it("keeps a real status and an opaque cursor", () => {
    expect(
      validateInvitationsSearch({
        status: "pending",
        cursor: "opaque-token-page-3",
      }),
    ).toEqual({ status: "pending", cursor: "opaque-token-page-3" })
  })

  it("drops an invalid status instead of throwing", () => {
    expect(validateInvitationsSearch({ status: "nope" })).toEqual({
      status: undefined,
      cursor: undefined,
    })
  })

  it("ignores a non-string cursor and an empty cursor", () => {
    expect(validateInvitationsSearch({ cursor: { nested: true } })).toEqual({
      status: undefined,
      cursor: undefined,
    })
    expect(validateInvitationsSearch({ cursor: "" })).toEqual({
      status: undefined,
      cursor: undefined,
    })
  })
})
