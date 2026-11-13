import { expect, it } from "vitest"
import { setEditedValue } from "../frontend/src/edits"
import { getPath, pruneEmpty, stableStringify } from "../frontend/src/path"

it.each([
  { original: undefined, inherited: true, changed: false, restored: true },
  { original: undefined, inherited: 12, changed: 20, restored: 12 },
  { original: undefined, inherited: ["a", "b"], changed: ["b"], restored: ["b", "a"] },
  { original: [], inherited: ["a"], changed: ["a"], restored: [] },
  { original: null, inherited: 12, changed: 20, restored: null },
  { original: true, inherited: true, changed: false, restored: true },
  { original: { enabled: true, soundId: 2 }, inherited: {}, changed: { enabled: false }, restored: { soundId: 2, enabled: true } },
])("restores the original override state after a round trip: $original / $inherited", ({ original, inherited, changed, restored }) => {
  const root: Record<string, unknown> = original === undefined ? {} : { value: structuredClone(original) }
  const before = stableStringify(root)
  setEditedValue(root, "value", changed, inherited, true)
  expect(stableStringify(pruneEmpty(root))).not.toBe(before)
  setEditedValue(root, "value", restored, inherited, true)
  expect(stableStringify(pruneEmpty(root))).toBe(before)
})

it("does not confuse explicit empty selections or uncapped values with inheritance", () => {
  const root = {}
  setEditedValue(root, "items", [], ["a"])
  setEditedValue(root, "cutoff", null, 12)
  expect(root).toEqual({ items: [], cutoff: null })
})

it("keeps the requested value when the inherited default changes during editing", () => {
  const root = {}
  setEditedValue(root, "cutoff", 30, 23)
  setEditedValue(root, "cutoff", 23, 40)
  expect(getPath(root, "cutoff")).toBe(23)
})

it("compares sparse style controls against their displayed defaults", () => {
  const root = {}
  const inherited = { preset: "rareTier2", text: "#ffffff", size: 45 }
  setEditedValue(root, "style", { preset: "rareTier2", size: 30 }, inherited)
  setEditedValue(root, "style", { preset: "rareTier2", size: 45 }, inherited)
  expect(getPath(root, "style")).toBeUndefined()
})

it("does not treat colour hex casing as a style change", () => {
  const root = {}
  setEditedValue(root, "style.text", "#123456", "#ABCDEF")
  setEditedValue(root, "style.text", "#abcdef", "#ABCDEF")
  expect(pruneEmpty(root)).toEqual({})
})
