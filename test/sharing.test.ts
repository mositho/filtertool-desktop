import { describe, expect, it, vi } from "vitest"

const byName = vi.fn().mockResolvedValue({ text: "{}", characters: 2 })
vi.mock("@wailsio/runtime", () => ({ Call: { ByName: byName }, Events: { On: vi.fn(() => vi.fn()) } }))

describe("native API boundary", () => {
  it("shares current filter data and the globals choice without a browser HTTP server", async () => {
    const { api } = await import("../frontend/src/api")
    await api.share("Caster", { buildProfile: { preferredColors: ["B"] }, buildSpecificOptions: {} }, true)
    expect(byName).toHaveBeenCalledWith("main.App.Dispatch", "share", {
      name: "Caster",
      config: { buildProfile: { preferredColors: ["B"] }, buildSpecificOptions: {} },
      includeGlobals: true,
    })
  })
  it("passes import payloads intact, including filter-local overrides", async () => {
    const { api } = await import("../frontend/src/api")
    const value = { format: "filtertool", config: { buildProfile: {}, buildSpecificOptions: {} }, defaults: { preferredColors: [] } }
    await api.importJSON("Imported", value)
    expect(byName).toHaveBeenCalledWith("main.App.Dispatch", "importJSON", { name: "Imported", value })
  })
})
