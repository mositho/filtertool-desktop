// @vitest-environment node
import { describe, expect, it } from "vitest"
import { updateGoPatch } from "../scripts/update-go-patch.mjs"

describe("Go patch maintenance", () => {
  const contents = "module example.com/test\n\ngo 1.26.9\n\nrequire example.com/library v1.0.0\n"

  it("selects the newest stable patch numerically without changing other module contents", () => {
    const releases = [
      { version: "go1.26.10", stable: true },
      { version: "go1.27.1", stable: true },
      { version: "go1.26.12rc1", stable: false },
      { version: "go1.26.11", stable: true },
      { version: "go1.26.8", stable: true },
    ]
    expect(updateGoPatch(contents, releases)).toBe(contents.replace("go 1.26.9", "go 1.26.11"))
  })

  it("does not downgrade or switch release lines when there is no newer patch", () => {
    expect(
      updateGoPatch(contents, [
        { version: "go1.26.8", stable: true },
        { version: "go1.27.1", stable: true },
      ]),
    ).toBe(contents)
    expect(updateGoPatch(contents, [])).toBe(contents)
  })

  it("rejects a missing or non-patch Go directive", () => {
    expect(() => updateGoPatch("module example.com/test\n", [])).toThrow("Go directive")
    expect(() => updateGoPatch("go 1.26\n", [])).toThrow("Go directive")
  })
})
