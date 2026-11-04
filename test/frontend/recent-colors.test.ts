import { afterEach, expect, it, vi } from "vitest"

afterEach(() => {
  localStorage.clear()
  vi.resetModules()
})

it("keeps twelve unique colors, promotes reused colors, and restores them in a new session", async () => {
  const { recentColors, rememberColor } = await import("../../frontend/src/recentColors")
  for (let i = 0; i < 14; i++) rememberColor(`#${i.toString(16).padStart(6, "0")}`)
  expect(recentColors.value).toHaveLength(12)
  expect(recentColors.value.at(-1)).toBe("#000002")
  rememberColor("#00000A")
  expect(recentColors.value[0]).toBe("#00000a")
  expect(recentColors.value).toHaveLength(12)
  vi.resetModules()
  const restored = await import("../../frontend/src/recentColors")
  expect(restored.recentColors.value).toEqual(recentColors.value)
})

it("ignores invalid saved colors and tolerates malformed optional history", async () => {
  localStorage.setItem("filtertool:recent-colors", JSON.stringify(["#AABBCC", "#aabbcc", null, "not a color", "#123456"]))
  const saved = await import("../../frontend/src/recentColors")
  expect(saved.recentColors.value).toEqual(["#aabbcc", "#123456"])
  vi.resetModules()
  localStorage.setItem("filtertool:recent-colors", "{")
  const malformed = await import("../../frontend/src/recentColors")
  expect(malformed.recentColors.value).toEqual([])
})
