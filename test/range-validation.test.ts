import { describe, expect, it } from "vitest"
import { filterRangeErrors } from "../src/config/range-validation"

describe("weapon phase validation", () => {
  it("requires an item class or base type target", () => {
    expect(filterRangeErrors({ buildProfile: { weaponPhases: [{ name: "Fast weapons", minAps: 1.4 }] } })).toContain(
      "Fast weapons: select at least one item class or base type.",
    )
    expect(filterRangeErrors({ buildProfile: { weaponPhases: [{ itemClasses: ["One Hand Axes"] }] } })).toEqual([])
    expect(filterRangeErrors({ buildProfile: { weaponPhases: [{ baseTypes: ["Rusted Hatchet"] }] } })).toEqual([])
  })

  it("uses the custom phase name in range errors", () => {
    expect(
      filterRangeErrors({
        buildProfile: { weaponPhases: [{ name: "Early axes", itemClasses: ["Axes"], minAreaLevel: 20, maxAreaLevel: 10 }] },
      }),
    ).toContain("Early axes: Min Area Level must not exceed Max Area Level.")
  })
})
