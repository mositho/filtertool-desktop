// @vitest-environment jsdom
import { afterEach, expect, it } from "vitest"
import { createApp, h, nextTick, reactive, type App } from "vue"
import ConfigForm from "../../frontend/src/components/ConfigForm.vue"
import type { FilterConfig, ReferenceData } from "../../frontend/src/api"
import referenceData from "../../internal/data/reference.json"
import styles from "../../internal/data/styles.json"
import { fieldStyleKeys, usesRarityText } from "../../frontend/src/namedStyles"
import { styleClipboard } from "../../frontend/src/styleClipboard"

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
function gemShortcut() {
  return root.querySelector<HTMLButtonElement>('[data-field-path="gemCallouts.gems"] [aria-haspopup="dialog"]')!
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
  ;[...root.querySelectorAll<HTMLButtonElement>("button")].find((button) => button.textContent?.trim() === "Expand all styles")!.click()
  await settle()
  root.scrollTop = 147
  gemShortcut().click()
  await settle()
  expect(dialog().textContent).toContain("Changes affect every rule in this filter")
  expect(dialog().querySelector<HTMLInputElement>('[aria-label="gemHighlight text"]')!.value).toBe("#abcdef")
  edit(dialog().querySelector('[aria-label="gemHighlight text"]')!, "#aabbcc")
  await settle()
  expect(state.localStyles?.gemHighlight).toEqual({ text: "#aabbcc" })
  expect(state.localStyles?.gemCallout).toBeUndefined()
  expect(root.querySelector<HTMLInputElement>('[aria-label="gemHighlight text"]')!.value).toBe("#aabbcc")
  expect(root.scrollTop).toBe(147)
  dialog().querySelector<HTMLButtonElement>('[title="Reset to global"]')!.click()
  await settle()
  expect(state.localStyles).toEqual({ currencyA: { border: "#123123" } })
  expect(dialog().querySelector<HTMLInputElement>('[aria-label="gemHighlight text"]')!.value).toBe("#123456")
  expect(root.querySelector<HTMLInputElement>('[aria-label="gemHighlight text"]')!.value).toBe("#123456")
  document.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true }))
  await settle()
  expect(dialog()).toBeNull()
  expect(document.activeElement).toBe(gemShortcut())
})

it("keeps edits sparse, removes the last override on reset, and dismisses on outside click", async () => {
  const state = mount()
  gemShortcut().click()
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
  const state = mount()
  gemShortcut().click()
  await settle()
  dialog().querySelector<HTMLButtonElement>('[title="Copy style"]')!.click()
  await settle()
  expect(styleClipboard.value?.style.size).toBe(45)
  dialog().querySelector<HTMLButtonElement>('[aria-label="Close style editor"]')!.click()
  await settle()
  root.querySelector<HTMLButtonElement>('[data-editor-group="styles"] > button')!.click()
  await settle()
  ;[...root.querySelectorAll<HTMLButtonElement>("button")].find((button) => button.textContent?.trim() === "Expand all styles")!.click()
  await settle()
  root.querySelector<HTMLButtonElement>('[aria-label="Paste currencyLow style"]')!.click()
  await settle()
  expect(state.localStyles?.currencyLow).toMatchObject({ size: 45, text: "#123456" })
  expect(root.querySelector<HTMLInputElement>('[aria-label="currencyLow size"]')!.value).toBe("45")
})
