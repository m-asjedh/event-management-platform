import { render, screen } from "@testing-library/react"
import { describe, expect, it } from "vitest"

import { ApiErrorView } from "@/components/api/ApiErrorView"
import { ApiError } from "@/lib/api/error"

describe("ApiErrorView", () => {
  it("surfaces the typed FORBIDDEN envelope", () => {
    render(
      <ApiErrorView
        error={
          new ApiError(403, {
            code: "FORBIDDEN",
            reason: "not allowed",
          })
        }
      />,
    )
    expect(screen.getByRole("heading", { name: "FORBIDDEN" })).toBeInTheDocument()
    expect(screen.getByText("not allowed")).toBeInTheDocument()
  })
})
