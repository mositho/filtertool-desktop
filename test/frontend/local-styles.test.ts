// @vitest-environment jsdom
import { afterEach, expect, it } from "vitest"
import { createApp, h, nextTick, reactive, type App } from "vue"
import ConfigForm from "../../frontend/src/components/ConfigForm.vue"
import type { FilterConfig, ReferenceData } from "../../frontend/src/api"
import referenceData from "../../internal/data/reference.json"
import styles from "../../internal/data/styles.json"
import { fieldStyleKeys, usesRarityText } from "../../frontend/src/namedStyles"
import { styleClipboard } from "../../frontend/src/styleClipboard"
import { editColor } from "./colorFields"

let app: App
let root: HTMLDivElement
afterEach(() => {
  app?.unmount()
  root?.remove()
  localStorage.clear()
  styleClipboard.value = null
})
async function settle() {
  await nextTick()
  await nextTick()
}
function mount(initial: FilterConfig = { buildProfile: {}, buildSpecificOptions: {} }) {
  localStorage.setItem("filtertool:collapsed:Style test", JSON.stringify({ "gems": false, "links": false, "links:areaLevels": false }))
  const state = reactive(initial)
  const globalStyles = { ...styles, gemHighlight: { ...styles.gemHighlight, text: "#123456" } }
  // The ordinary styles response includes saved local edits; globalStyles must not.
  const reference = {
    ...referenceData,
    globalStyles,
    styles: { ...globalStyles, gemHighlight: { ...globalStyles.gemHighlight, ...(initial.localStyles?.gemCallout as object) } },
    sounds: [],
  } as unknown as ReferenceData
  root = document.createElement("div")
  document.body.append(root)
  app = createApp({ setup: () => () => h(ConfigForm, { modelValue: state, reference, filterName: "Style test" }) })
  app.mount(root)
  return state
}
function gemShortcut(field = "gems") {
  return root.querySelector<HTMLButtonElement>(`[data-field-path="gemCallouts.${field}"] [aria-haspopup="dialog"]`)!
}
function dialog() {
  return document.querySelector<HTMLElement>('[role="dialog"]')!
}
function edit(input: HTMLInputElement, value: string) {
  input.value = value
  input.dispatchEvent(new Event("input", { bubbles: true }))
}

it("adds Styles at the end without writing overrides just by viewing the filter", () => {
  const state = mount()
  expect([...root.querySelectorAll("[data-editor-group]")].at(-1)?.getAttribute("data-editor-group")).toBe("styles")
  expect(state.localStyles).toBeUndefined()
})

it.each(["link", "preview"])("opens and focuses the matching Styles entry from the popup %s without changing the filter", async (entry) => {
  const state = mount()
  gemShortcut().click()
  await settle()
  const selector =
    entry === "link" ? '[aria-label="Show gemHighlight in Styles"]' : '[aria-label="Show gemHighlight in Styles (gemHighlight preview)"]'
  dialog().querySelector<HTMLButtonElement>(selector)!.click()
  await settle()
  expect(dialog()).toBeNull()
  expect(root.querySelector('[data-editor-group="styles"] > button')!.getAttribute("aria-expanded")).toBe("true")
  expect(document.activeElement).toBe(root.querySelector('[data-editor-group="styles"] [data-style-key="gemHighlight"]'))
  expect(state).toEqual({ buildProfile: {}, buildSpecificOptions: {} })
})

it("edits in place, synchronizes the bottom panel, and resets to global rather than stale saved styles", async () => {
  const state = mount({
    buildProfile: {},
    buildSpecificOptions: {},
    localStyles: { gemCallout: { text: "#abcdef" }, currencyA: { border: "#123123" } },
  })
  root.querySelector<HTMLButtonElement>('[data-editor-group="styles"] > button')!.click()
  await settle()
  ;[...root.querySelectorAll<HTMLButtonElement>('[data-editor-group="styles"] button')]
    .find((button) => button.textContent?.trim() === "Expand all")!
    .click()
  await settle()
  root.scrollTop = 147
  gemShortcut().click()
  await settle()
  expect(dialog().textContent).toContain("Changes affect every rule in this filter")
  expect(dialog().querySelector<HTMLButtonElement>('[aria-label="gemHighlight text"]')!.value).toBe("#abcdef")
  dialog().querySelector<HTMLButtonElement>('[aria-label="gemHighlight text"]')!.click()
  await settle()
  const color = dialog().querySelector<HTMLInputElement>('[aria-label="gemHighlight text hex"]')!
  color.value = "#aabbcc"
  color.dispatchEvent(new Event("input", { bubbles: true }))
  await settle()
  expect(state.localStyles?.gemHighlight).toEqual({ text: "#aabbcc" })
  expect(state.localStyles?.gemCallout).toBeUndefined()
  expect(root.querySelector<HTMLButtonElement>('[aria-label="gemHighlight text"]')!.value).toBe("#aabbcc")
  expect(root.scrollTop).toBe(147)
  // Escape closes only the nested picker, not the surrounding style editor.
  color.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true }))
  await settle()
  expect(dialog()).not.toBeNull()
  expect(dialog().querySelector('[aria-label="gemHighlight text hex"]')).toBeNull()
  dialog().querySelector<HTMLButtonElement>('[title="Reset to global"]')!.click()
  await settle()
  expect(state.localStyles).toEqual({ currencyA: { border: "#123123" } })
  expect(dialog().querySelector<HTMLButtonElement>('[aria-label="gemHighlight text"]')!.value).toBe("#123456")
  expect(root.querySelector<HTMLButtonElement>('[aria-label="gemHighlight text"]')!.value).toBe("#123456")
  document.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true }))
  await settle()
  expect(dialog()).toBeNull()
  expect(document.activeElement).toBe(gemShortcut())
})

it.each(["gems", "styleOnly"])("edits the highlight style for %s, resets it, and dismisses on outside click", async (field) => {
  const state = mount()
  gemShortcut(field).click()
  await settle()
  edit(dialog().querySelector('[aria-label="gemHighlight opacity"]')!, "0.5")
  await settle()
  expect(state.localStyles).toEqual({ gemHighlight: { backgroundOpacity: 0.5 } })
  dialog().querySelector<HTMLButtonElement>('[title="Reset to global"]')!.click()
  await settle()
  expect(state.localStyles).toBeUndefined()
  document.body.dispatchEvent(new MouseEvent("mousedown", { bubbles: true }))
  await settle()
  expect(dialog()).toBeNull()
})

it("previews ordinary rarity colors without offering highlight styles in a link card", async () => {
  mount()
  root.querySelector<HTMLButtonElement>('[data-field-path="links.threeLinkMaxAreaLevel"] [aria-haspopup="dialog"]')!.click()
  await settle()
  expect(dialog().querySelectorAll("[data-rarity-preview]")).toHaveLength(3)
  expect(dialog().querySelector('[aria-label="threeLink text"]')).toBeNull()
  expect(dialog().querySelector('[aria-label="Style to edit"]')).toBeNull()
  expect(dialog().querySelector<HTMLElement>('[data-rarity-preview="Magic"]')!.style.color).toBe("rgb(136, 136, 255)")
})

it.each(["Normal", "Magic", "Rare"])("inherits %s text for its equipment highlight preview", async (rarity) => {
  const key = `highlightedEquipment${rarity}`
  mount({
    buildProfile: {},
    buildSpecificOptions: {},
    localStyles: { [rarity.toLowerCase()]: { text: "#123456" }, [key]: { text: "#ff0000" } },
  })
  root.querySelector<HTMLButtonElement>('[data-editor-group="styles"] > button')!.click()
  await settle()
  const equipment = [...root.querySelectorAll<HTMLButtonElement>('[data-editor-group="styles"] button')].find((button) =>
    button.textContent?.includes("Equipment"),
  )!
  equipment.click()
  await settle()
  const row = root.querySelector<HTMLElement>(`[data-style-key="${key}"]`)!
  expect(row).not.toBeNull()
  expect(row.textContent).toContain("Rarities")
  expect(row.querySelector(`[aria-label="${key} text"]`)).toBeNull()
  expect(row.querySelectorAll("[data-rarity-preview]")).toHaveLength(1)
  expect(row.querySelector<HTMLElement>(`[data-rarity-preview="${rarity}"]`)!.style.color).toBe("rgb(18, 52, 86)")
})

it("maps ordinary rarity rules to their own styles", () => {
  expect(fieldStyleKeys({ path: "leagueMechanic.rules.charts.baseTypes", label: "Charts", control: "multiselect" })).toEqual([
    "leagueMechanic",
  ])
  expect(fieldStyleKeys({ path: "magicItems.bigMaxAreaLevel", label: "Big", control: "number" })).toEqual(["magic"])
  expect(fieldStyleKeys({ path: "normalItems.maxAreaLevel", label: "Normal", control: "number" })).toEqual(["normal"])
  expect(fieldStyleKeys({ path: "early.twoSocketMaxAreaLevel", label: "Two Sockets", control: "number" })).toEqual([
    "normal",
    "magic",
    "rare",
  ])
  expect(fieldStyleKeys({ path: "early.earlyMaxAreaLevel", label: "Early", control: "number" })).toEqual([
    "normal",
    "magic",
    "rare",
    "earlyShieldBase",
  ])
  expect(usesRarityText("fiveLink")).toBe(true)
  expect(usesRarityText("sixSocket")).toBe(false)
})

it("shares complete style copies between cards and the bottom panel without retaining the target's old size", async () => {
  const state = mount({
    buildProfile: {},
    buildSpecificOptions: {},
    localStyles: { gemHighlight: { size: 32 }, currencyLow: { size: 41 } },
  })
  gemShortcut().click()
  await settle()
  dialog().querySelector<HTMLButtonElement>('[title="Copy style"]')!.click()
  await settle()
  expect(styleClipboard.value?.style.size).toBe(32)
  dialog().querySelector<HTMLButtonElement>('[aria-label="Close style editor"]')!.click()
  await settle()
  root.querySelector<HTMLButtonElement>('[data-editor-group="styles"] > button')!.click()
  await settle()
  ;[...root.querySelectorAll<HTMLButtonElement>('[data-editor-group="styles"] button')]
    .find((button) => button.textContent?.trim() === "Expand all")!
    .click()
  await settle()
  root.querySelector<HTMLButtonElement>('[aria-label="Paste Low-value Currency style"]')!.click()
  await settle()
  expect(state.localStyles?.currencyLow).toMatchObject({ size: 32, text: "#123456" })
  expect(root.querySelector<HTMLInputElement>('[aria-label="Low-value Currency size"]')!.value).toBe("32")
})

it.each([
  { tier: "s", key: "currencyA", label: "Currency · S Tier" },
  { tier: "a", key: "currencyB", label: "Currency · A Tier" },
  { tier: "b", key: "currencyC", label: "Currency · B Tier" },
])("uses the $tier tier label in its style popup and Styles panel while preserving saved overrides", async ({ tier, key, label }) => {
  const state = mount()
  root.querySelector<HTMLButtonElement>(`[data-currency-tier="${tier}"] [title="Edit style for this filter"]`)!.click()
  await settle()
  expect(dialog().getAttribute("aria-label")).toBe(`Edit ${label} style`)
  expect(dialog().textContent).toContain(label)
  expect(dialog().textContent).not.toContain(key)
  await editColor(dialog().querySelector<HTMLButtonElement>(`[aria-label="${label} text"]`)!, "#123abc")
  expect(state.localStyles).toEqual({ [key]: { text: "#123abc" } })
  dialog().querySelector<HTMLButtonElement>(`[aria-label="Show ${label} in Styles"]`)!.click()
  await settle()
  const row = root.querySelector<HTMLElement>(`[data-editor-group="styles"] [data-style-key="${key}"]`)!
  expect(document.activeElement).toBe(row)
  expect(row.textContent).toContain(label)
  expect(row.textContent).not.toContain(key)
})
