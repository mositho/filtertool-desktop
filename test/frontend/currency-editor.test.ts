// @vitest-environment jsdom
import { afterEach, expect, it, vi } from "vitest"
import { createApp, nextTick, reactive, type App } from "vue"
import CurrencyEditor from "../../frontend/src/components/CurrencyEditor.vue"
import type { ReferenceData } from "../../frontend/src/api"
import { getPath } from "../../frontend/src/path"
import FieldInput from "../../frontend/src/components/FieldInput.vue"
import ConfigForm from "../../frontend/src/components/ConfigForm.vue"
import builtinReference from "../../internal/data/reference.json"
import soundManifest from "../../internal/data/manifest.json"
import { BUILD_SPECIFIC_SCHEMA } from "../../src/config/form-schema"
import { LEAGUE_RULES } from "../../src/config/currency"

vi.mock("../../frontend/src/api", () => ({
  api: {
    soundUrl: async () => {
      throw new Error("No native audio preview in tests")
    },
  },
}))

let app: App | undefined
let container: HTMLDivElement
afterEach(() => {
  app?.unmount()
  container?.remove()
  localStorage.clear()
})

function mount(defaults: Record<string, unknown> = {}, filterName = "currency-test") {
  container = document.createElement("div")
  document.body.append(container)
  const root = reactive<Record<string, unknown>>({})
  const reference = {
    baseTypesByClass: { "Stackable Currency": ["Scroll Fragment", "Chaos Orb", "Chromatic Orb", "Cursed Poppet", "Portal Scroll"] },
    sounds: soundManifest.map((sound) => ({ ...sound, hasFile: true })),
    currencySounds: {
      "Chromatic Orb": { enabled: true, tts: "chromatic_orb" },
      "Chaos Orb": { enabled: true, tts: "chaos_orb" },
    },
    currencyIcons: {
      "Chromatic Orb": { enabled: true, iconShape: "Pentagon", iconColor: "Purple", iconSize: 2 },
      "Chaos Orb": { enabled: true, iconShape: "Circle", iconColor: "Orange", iconSize: 2 },
    },
  } as unknown as ReferenceData
  app = createApp(CurrencyEditor, { root, reference, defaults, filterName })
  app.mount(container)
  return root
}

async function select(name: string) {
  item(name).click()
  await nextTick()
}

function item(name: string): HTMLButtonElement {
  return [...container.querySelectorAll<HTMLButtonElement>("[data-currency-item]")].find((button) => button.dataset.currencyItem === name)!
}

async function click(text: string) {
  ;[...container.querySelectorAll<HTMLButtonElement>("button")].find((button) => button.textContent?.trim() === text)!.click()
  await nextTick()
}

async function setCurrencyVisibility(value: string) {
  const select = container.querySelector<HTMLSelectElement>('[aria-label="Currency visibility"]')!
  select.value = value
  select.dispatchEvent(new Event("change", { bubbles: true }))
  await nextTick()
}

async function resetCurrency() {
  container.querySelector<HTMLButtonElement>('button[aria-label="Reset selected currency"]')!.click()
  await nextTick()
}

it("opens tier fallback details without tier previews or item counts", async () => {
  const root = mount()
  const card = container.querySelector('[data-currency-tier="s"]')!
  expect(card.querySelector("[data-tier-information]")).toBeNull()
  expect(card.querySelector('[title="Style preview"]')).toBeNull()
  expect(card.querySelector('[aria-label="S Tier visibility"]')).not.toBeNull()
  card.querySelector<HTMLButtonElement>('[aria-label="S Tier settings"]')!.click()
  await nextTick()
  const panel = container.querySelector('[aria-label="Currency item settings"]')!
  expect(panel.querySelector("h3")?.textContent).toContain("S Tier")
  expect(tierHeader("s").getAttribute("aria-expanded")).toBe("false")
  expect(panel.querySelector('[aria-label="Currency tier"]')).toBeNull()
  const field = panel.querySelector('[data-field-path="currency.tiers.s.sound"]')!
  const mode = field.querySelector<HTMLSelectElement>("select")!
  expect(mode.value).toBe("alert")
  expect(field.querySelectorAll("select")[1].value).toBe("6")
  mode.value = "none"
  mode.dispatchEvent(new Event("change", { bubbles: true }))
  await nextTick()
  expect(getPath(root, "currency.tiers.s.sound.enabled")).toBe(false)
  panel.querySelector<HTMLButtonElement>('[aria-label="Reset S Tier settings"]')!.click()
  await nextTick()
  expect(getPath(root, "currency.tiers.s.sound")).toBeUndefined()
  expect(mode.value).toBe("alert")
})

it("shows actual inherited item icons and preserves item TTS over tier fallback choices", async () => {
  const root = mount({
    currency: { tiers: { b: { sound: { soundId: 8 }, icon: { enabled: true, iconShape: "Square", iconColor: "Red", iconSize: 0 } } } },
  })
  await select("Chromatic Orb")
  const icon = container.querySelector('[data-field-path="currency.items.Chromatic Orb.icon"]')!
  expect(icon.querySelector('[title="Pentagon"]')?.classList.contains("border-white")).toBe(true)
  expect(icon.querySelector('[title="Purple"]')?.classList.contains("border-white")).toBe(true)
  expect(icon.textContent).not.toContain("Default icon")
  const sound = container.querySelector<HTMLSelectElement>('[data-field-path="currency.items.Chromatic Orb.sound"] select')!
  expect(sound.value).toBe("preset")
  icon.querySelector<HTMLButtonElement>('[title="No Icon"]')!.click()
  sound.value = "none"
  sound.dispatchEvent(new Event("change", { bubbles: true }))
  await nextTick()
  expect(getPath(root, "currency.items.Chromatic Orb.icon.enabled")).toBe(false)
  expect(getPath(root, "currency.items.Chromatic Orb.sound.enabled")).toBe(false)
  icon.querySelector<HTMLButtonElement>('button[title="Use default"]')!.click()
  await nextTick()
  expect(icon.querySelector('[title="Pentagon"]')?.classList.contains("border-white")).toBe(true)
})

it("moves selected currency once, resets to shared defaults, and excludes league items and scrolls", async () => {
  const root = mount({ currency: { assignments: { "Scroll Fragment": "b" } } })
  expect(container.textContent).not.toContain("Cursed Poppet")
  expect(container.textContent).not.toContain("Portal Scroll")
  await select("Scroll Fragment")
  const tier = container.querySelector<HTMLSelectElement>('[aria-label="Currency tier"]')!
  expect(tier.value).toBe("b")
  tier.value = "s"
  tier.dispatchEvent(new Event("change", { bubbles: true }))
  await nextTick()
  expect(getPath(root, "currency.assignments.Scroll Fragment")).toBe("s")
  expect(tier.value).toBe("s")
  expect(container.querySelectorAll('[data-currency-item="Scroll Fragment"]')).toHaveLength(1)
  await select("Scroll Fragment")
  const reset = container.querySelector<HTMLButtonElement>('button[aria-label="Reset selected currency"]')!
  expect(reset.parentElement?.querySelector("h3")?.textContent).toContain("Scroll Fragment")
  expect(reset.closest('[aria-label="Currency item settings"]')).toBe(tier.parentElement)
  expect(reset.querySelector("svg")).not.toBeNull()
  expect(reset.textContent?.trim()).toBe("")
  expect(reset.title).toBe("Reset tier, visibility, sound, icon, and stack settings to defaults.")
  reset.click()
  await nextTick()
  expect(getPath(root, "currency.assignments.Scroll Fragment")).toBeUndefined()
  expect(tier.value).toBe("b")
  expect(item("Scroll Fragment").closest("section")?.getAttribute("aria-label")).toBe("B Tier")
})

it("shows only the current selection's tier and changes it immediately", async () => {
  const root = mount()
  expect(container.querySelector<HTMLSelectElement>('[aria-label="Currency tier"]')?.disabled).toBe(true)
  expect(container.textContent).not.toContain("Move to tier")
  await select("Chromatic Orb")
  const tier = container.querySelector<HTMLSelectElement>('[aria-label="Currency tier"]')!
  expect(tier.closest("label")).toBeNull()
  expect(tier.value).toBe("b")
  await select("Chaos Orb")
  expect(tier.value).toBe("a")
  expect(root).toEqual({})
  tier.value = "s"
  tier.dispatchEvent(new Event("change", { bubbles: true }))
  await nextTick()
  expect(getPath(root, "currency.assignments.Chaos Orb")).toBe("s")
  expect(item("Chaos Orb").closest("section")?.getAttribute("data-currency-tier")).toBe("s")
  item("Chaos Orb").dispatchEvent(new Event("dragstart", { bubbles: true }))
  container.querySelector('[data-currency-tier="low"]')!.dispatchEvent(new Event("drop", { bubbles: true, cancelable: true }))
  await nextTick()
  expect(tier.value).toBe("low")
})

it("shows mixed tiers for a multi-selection and moves them immediately from the dropdown", async () => {
  const root = mount()
  await select("Scroll Fragment")
  item("Chaos Orb").dispatchEvent(new MouseEvent("click", { bubbles: true, ctrlKey: true }))
  await nextTick()
  const tier = container.querySelector<HTMLSelectElement>('[aria-label="Currency tier"]')!
  expect(tier.selectedOptions[0].textContent).toBe("Mixed tiers")
  expect(container.querySelector('[aria-label="Reset selected currency"] svg')).not.toBeNull()
  tier.value = "b"
  tier.dispatchEvent(new Event("change", { bubbles: true }))
  await nextTick()
  expect(getPath(root, "currency.assignments")).toEqual({ "Scroll Fragment": "b", "Chaos Orb": "b" })
  expect(tier.value).toBe("b")
})

it("hides individual currency and restores tier visibility without changing assignment", async () => {
  const root = mount()
  await select("Scroll Fragment")
  await setCurrencyVisibility("Hide")
  expect(getPath(root, "currency.items.Scroll Fragment.visibility")).toBe("Hide")
  expect(item("Scroll Fragment").textContent).toContain("Hidden")
  await resetCurrency()
  expect(getPath(root, "currency.items.Scroll Fragment.visibility")).toBeUndefined()
  expect(getPath(root, "currency.assignments")).toBeUndefined()
})

it("searches and bulk-selects only matching currency", async () => {
  const root = mount()
  const search = container.querySelector<HTMLInputElement>('[aria-label="Search currency"]')!
  search.value = "fragment"
  search.dispatchEvent(new Event("input", { bubbles: true }))
  await nextTick()
  await click("Select results")
  await setCurrencyVisibility("Hide")
  expect(getPath(root, "currency.items.Scroll Fragment.visibility")).toBe("Hide")
  expect(getPath(root, "currency.items.Chaos Orb")).toBeUndefined()
})

it("renders tier cards and opens item-specific controls without style editors", async () => {
  const root = mount()
  expect(container.querySelectorAll("[data-currency-tier]")).toHaveLength(5)
  expect(root).toEqual({})
  expect(container.querySelector('[aria-label="Currency item settings"]')?.textContent).toContain("Select Currency")
  await select("Scroll Fragment")
  expect(container.querySelector('[aria-label="Currency item settings"]')?.textContent).toContain("Scroll Fragment")
  expect(container.querySelector('[data-field-path="currency.items.Scroll Fragment.sound"]')).not.toBeNull()
  expect(container.querySelector('[data-field-path="currency.items.Scroll Fragment.icon"]')).not.toBeNull()
  expect(container.querySelector('[data-field-path$=".style"]')).toBeNull()
  expect(container.textContent).not.toContain("Moving keeps")
  expect(container.textContent).not.toContain("Drag items to another tier")
  expect(container.textContent).not.toContain("Select a currency to edit")
  for (const tier of container.querySelectorAll("[data-currency-tier]")) {
    expect([...tier.querySelectorAll("select option")].map((option) => option.textContent)).toEqual(["Show", "Hide"])
  }
})

it("keeps the detail editor visible when selecting and clearing currencies", async () => {
  const root = mount()
  const panel = container.querySelector('[aria-label="Currency item settings"]')!
  expect(panel.textContent).toContain("Select Currency")
  const tier = container.querySelector<HTMLSelectElement>('[aria-label="Currency tier"]')!
  const visibility = container.querySelector<HTMLSelectElement>('[aria-label="Currency visibility"]')!
  const reset = container.querySelector<HTMLButtonElement>('[aria-label="Reset selected currency"]')!
  expect(tier.selectedOptions[0].textContent).toBe("—")
  expect(visibility.selectedOptions[0].textContent).toBe("—")
  expect(tier.disabled && visibility.disabled && reset.disabled).toBe(true)
  expect(panel.querySelectorAll("[data-field-path]")).toHaveLength(0)
  await select("Chromatic Orb")
  expect(tier.disabled || visibility.disabled || reset.disabled).toBe(false)
  expect(tier.value).toBe("b")
  expect(visibility.value).toBe("Show")
  expect(container.querySelector('[aria-label="Currency item settings"]')).toBe(panel)
  expect(panel.textContent).toContain("Chromatic Orb")
  item("Chaos Orb").dispatchEvent(new MouseEvent("click", { bubbles: true, ctrlKey: true }))
  await nextTick()
  expect(panel.textContent).toContain("2 currencies selected")
  await click("Clear selection")
  expect(container.querySelector('[aria-label="Currency item settings"]')).toBe(panel)
  expect(panel.textContent).toContain("Select Currency")
  expect(container.querySelector('[aria-label="Currency tier"]')).toBe(tier)
  expect(container.querySelector('[aria-label="Currency visibility"]')).toBe(visibility)
  expect(tier.selectedOptions[0].textContent).toBe("—")
  expect(visibility.selectedOptions[0].textContent).toBe("—")
  expect(tier.disabled && visibility.disabled && reset.disabled).toBe(true)
  expect(panel.querySelectorAll("[data-field-path]")).toHaveLength(0)
  expect(root).toEqual({})
})

function tierHeader(key: string): HTMLButtonElement {
  return container.querySelector<HTMLButtonElement>(`[data-currency-tier="${key}"] button[aria-expanded]`)!
}

function tierBody(key: string): HTMLElement {
  return document.getElementById(tierHeader(key).getAttribute("aria-controls")!)!
}

it("collapses tiers without closing or shrinking the reserved detail column", async () => {
  const root = mount()
  await select("Chromatic Orb")
  const panel = container.querySelector('[aria-label="Currency item settings"]')!
  const aside = container.querySelector('aside[aria-label="Currency details"]')!
  expect(aside.classList.contains("@min-[48rem]:min-h-[28rem]")).toBe(true)
  expect(aside.parentElement?.classList.contains("@min-[48rem]:grid-cols-[minmax(0,1fr)_20rem]")).toBe(true)
  await click("Collapse tiers")
  for (const key of ["s", "a", "b", "leaguestart", "low"]) {
    expect(tierHeader(key).getAttribute("aria-expanded")).toBe("false")
    expect(tierBody(key).style.display).toBe("none")
  }
  expect(container.querySelector('[aria-label="Currency item settings"]')).toBe(panel)
  expect(panel.closest('[style*="display: none"]')).toBeNull()
  expect(root).toEqual({})
  await click("Expand tiers")
  expect(tierHeader("b").getAttribute("aria-expanded")).toBe("true")
  expect(tierBody("b").style.display).not.toBe("none")
  expect(container.querySelector('[aria-label="Currency item settings"]')).toBe(panel)
})

it("keeps the Show/Hide control independent of tier collapse", async () => {
  const root = mount()
  const visibility = container.querySelector<HTMLSelectElement>('[aria-label="A Tier visibility"]')!
  visibility.value = "Hide"
  visibility.dispatchEvent(new Event("change", { bubbles: true }))
  await nextTick()
  expect(getPath(root, "currency.tiers.a.visibility")).toBe("Hide")
  expect(tierHeader("a").getAttribute("aria-expanded")).toBe("false")
  tierHeader("a").click()
  await nextTick()
  expect(visibility.value).toBe("Hide")
  expect(tierHeader("a").getAttribute("aria-expanded")).toBe("true")
})

it("remembers tier collapse per filter without editing the filter configuration", async () => {
  const root = mount({}, "first")
  tierHeader("a").click()
  await nextTick()
  expect(root).toEqual({})
  expect(JSON.parse(localStorage.getItem("filtertool:currency-tiers:first")!)).toMatchObject({ a: false })
  app!.unmount()
  container.remove()
  mount({}, "second")
  expect(tierHeader("a").getAttribute("aria-expanded")).toBe("false")
  app!.unmount()
  container.remove()
  mount({}, "first")
  expect(tierHeader("a").getAttribute("aria-expanded")).toBe("true")
})

it("accepts drops on collapsed tier headers and keeps the item's detail panel open", async () => {
  const root = mount()
  await select("Chaos Orb")
  const panel = container.querySelector('[aria-label="Currency item settings"]')!
  item("Chaos Orb").dispatchEvent(new Event("dragstart", { bubbles: true }))
  tierHeader("s").dispatchEvent(new Event("dragover", { bubbles: true, cancelable: true }))
  tierHeader("s").dispatchEvent(new Event("drop", { bubbles: true, cancelable: true }))
  await nextTick()
  expect(getPath(root, "currency.assignments.Chaos Orb")).toBe("s")
  expect(tierHeader("s").getAttribute("aria-expanded")).toBe("false")
  expect(tierBody("s").style.display).toBe("none")
  expect(container.querySelector('[aria-label="Currency item settings"]')).toBe(panel)
  expect(panel.textContent).toContain("Chaos Orb")
})

it("reveals search matches in collapsed tiers without changing the saved collapse state", async () => {
  const root = mount()
  const saved = localStorage.getItem("filtertool:currency-tiers:currency-test")
  const search = container.querySelector<HTMLInputElement>('[aria-label="Search currency"]')!
  search.value = "Chaos"
  search.dispatchEvent(new Event("input", { bubbles: true }))
  await nextTick()
  expect(tierHeader("a").getAttribute("aria-expanded")).toBe("true")
  tierHeader("a").click()
  await nextTick()
  expect(tierHeader("a").getAttribute("aria-expanded")).toBe("false")
  search.value = ""
  search.dispatchEvent(new Event("input", { bubbles: true }))
  await nextTick()
  expect(tierHeader("a").getAttribute("aria-expanded")).toBe("false")
  expect(localStorage.getItem("filtertool:currency-tiers:currency-test")).toBe(saved)
  expect(root).toEqual({})
})

it("uses simple Show/Hide values for items while reset restores tier visibility", async () => {
  const root = mount({ currency: { tiers: { low: { visibility: "Hide" } } } })
  await select("Scroll Fragment")
  const visibility = container.querySelector<HTMLSelectElement>('[aria-label="Currency visibility"]')!
  expect(visibility.value).toBe("Hide")
  expect([...visibility.options].map((option) => option.textContent)).toEqual(["Show", "Hide"])
  visibility.value = "Show"
  visibility.dispatchEvent(new Event("change", { bubbles: true }))
  await nextTick()
  expect(getPath(root, "currency.items.Scroll Fragment.visibility")).toBe("Show")
  await resetCurrency()
  expect(visibility.value).toBe("Hide")
  expect(getPath(root, "currency.items.Scroll Fragment.visibility")).toBeUndefined()
})

it("shows a dash for mixed visibility and applies the chosen state to the entire selection", async () => {
  const root = mount({ currency: { tiers: { b: { visibility: "Hide" } } } })
  expect(container.querySelector<HTMLSelectElement>('[aria-label="Currency visibility"]')?.disabled).toBe(true)
  await select("Chromatic Orb")
  const visibility = container.querySelector<HTMLSelectElement>('[aria-label="Currency visibility"]')!
  expect(visibility.value).toBe("Hide")
  item("Chaos Orb").dispatchEvent(new MouseEvent("click", { bubbles: true, ctrlKey: true }))
  await nextTick()
  expect(visibility.value).toBe("")
  expect(visibility.selectedOptions[0].textContent).toBe("—")
  expect(visibility.selectedOptions[0].disabled).toBe(true)
  await setCurrencyVisibility("Show")
  expect(visibility.value).toBe("Show")
  expect(getPath(root, "currency.items.Chromatic Orb.visibility")).toBe("Show")
  expect(getPath(root, "currency.items.Chaos Orb.visibility")).toBeUndefined()
  await setCurrencyVisibility("Hide")
  expect(visibility.value).toBe("Hide")
  expect(getPath(root, "currency.items.Chromatic Orb.visibility")).toBeUndefined()
  expect(getPath(root, "currency.items.Chaos Orb.visibility")).toBe("Hide")
  await resetCurrency()
  expect(visibility.selectedOptions[0].textContent).toBe("—")
  expect(getPath(root, "currency.items.Chromatic Orb.visibility")).toBeUndefined()
  expect(getPath(root, "currency.items.Chaos Orb.visibility")).toBeUndefined()
  expect(container.textContent).not.toContain("Reset visibility")
  expect(container.querySelectorAll('aside[aria-label="Currency details"] select[aria-label="Currency visibility"]')).toHaveLength(1)
})

it("drags a multi-selection to an empty tier and keeps individual sound and stack overrides", async () => {
  const root = mount()
  await select("Scroll Fragment")
  const stack = container.querySelector<HTMLInputElement>('[data-field-path="currency.items.Scroll Fragment.minStackSize"] input')!
  stack.value = "5"
  stack.dispatchEvent(new Event("input", { bubbles: true }))
  const sound = container.querySelector<HTMLSelectElement>('[data-field-path="currency.items.Scroll Fragment.sound"] select')!
  sound.value = "alert"
  sound.dispatchEvent(new Event("change", { bubbles: true }))
  await nextTick()
  item("Chaos Orb").dispatchEvent(new MouseEvent("click", { bubbles: true, ctrlKey: true }))
  await nextTick()
  expect(container.querySelectorAll('[data-currency-item][aria-pressed="true"]')).toHaveLength(2)
  item("Scroll Fragment").dispatchEvent(new Event("dragstart", { bubbles: true }))
  const target = container.querySelector('[data-currency-tier="s"]')!
  const over = new Event("dragover", { bubbles: true, cancelable: true })
  target.dispatchEvent(over)
  expect(over.defaultPrevented).toBe(true)
  target.dispatchEvent(new Event("drop", { bubbles: true, cancelable: true }))
  await nextTick()
  expect(getPath(root, "currency.assignments")).toEqual({ "Scroll Fragment": "s", "Chaos Orb": "s" })
  expect(getPath(root, "currency.items.Scroll Fragment.minStackSize")).toBe(5)
  expect(getPath(root, "currency.items.Scroll Fragment.sound.soundId")).toBe(1)
  expect(target.querySelectorAll("[data-currency-item]")).toHaveLength(2)
})

it("ignores external drops and cancelled drags", async () => {
  const root = mount()
  const target = container.querySelector('[data-currency-tier="s"]')!
  target.dispatchEvent(new Event("drop", { bubbles: true, cancelable: true }))
  expect(root).toEqual({})
  item("Chaos Orb").dispatchEvent(new Event("dragstart", { bubbles: true }))
  item("Chaos Orb").dispatchEvent(new Event("dragend", { bubbles: true }))
  target.dispatchEvent(new Event("drop", { bubbles: true, cancelable: true }))
  expect(root).toEqual({})
})

it("updates an item's sound without changing another currency and resets its overrides", async () => {
  const root = mount({ currency: { tiers: { a: { sound: { tts: "Inherited TTS" } } } } })
  await select("Chaos Orb")
  const sound = container.querySelector<HTMLSelectElement>('[data-field-path="currency.items.Chaos Orb.sound"] select')!
  sound.value = "none"
  sound.dispatchEvent(new Event("change", { bubbles: true }))
  await nextTick()
  expect(getPath(root, "currency.items.Chaos Orb.sound")).toEqual({ enabled: false })
  expect(getPath(root, "currency.items.Scroll Fragment")).toBeUndefined()
  container.querySelector<HTMLButtonElement>('[data-field-path="currency.items.Chaos Orb.sound"] button[title="Use default"]')!.click()
  await nextTick()
  expect(getPath(root, "currency.items.Chaos Orb.sound")).toBeUndefined()
})

it("shows Chromatic Orb's actual built-in TTS template without writing an override", async () => {
  const root = mount()
  await select("Chromatic Orb")
  const field = container.querySelector('[data-field-path="currency.items.Chromatic Orb.sound"]')!
  const controls = field.querySelectorAll<HTMLSelectElement>("select")
  expect(controls[0].selectedOptions[0].textContent).toBe("TTS Template")
  expect(controls[1].value).toBe("chromatic_orb")
  expect(controls[1].selectedOptions[0].textContent).toBe("Chromatic Orb")
  expect(field.textContent).not.toContain("Inherited item callout")
  expect(root).toEqual({})
})

it.each(["preset", "tts", "file"])("keeps the %s sound editor open after switching from None", async (mode) => {
  const root = mount()
  await select("Chromatic Orb")
  const field = container.querySelector('[data-field-path="currency.items.Chromatic Orb.sound"]')!
  const type = field.querySelector<HTMLSelectElement>("select")!
  type.value = "none"
  type.dispatchEvent(new Event("change", { bubbles: true }))
  await nextTick()
  type.value = mode
  type.dispatchEvent(new Event("change", { bubbles: true }))
  await nextTick()
  expect(type.value).toBe(mode)
  if (mode === "preset") {
    const templates = field.querySelectorAll<HTMLSelectElement>("select")[1]
    templates.value = "chromatic_orb"
    templates.dispatchEvent(new Event("change", { bubbles: true }))
  } else {
    const input = field.querySelector<HTMLInputElement>("input")!
    input.value = mode === "tts" ? "My chromatic callout" : "chromatic-custom.mp3"
    input.dispatchEvent(new Event("input", { bubbles: true }))
  }
  await nextTick()
  expect(type.value).toBe(mode)
  const sound = getPath(root, "currency.items.Chromatic Orb.sound")
  expect(sound).toEqual(
    mode === "file"
      ? { enabled: true, soundFileName: "chromatic-custom.mp3" }
      : mode === "preset"
        ? undefined
        : { enabled: true, tts: "My chromatic callout" },
  )
})

it("keeps an empty template picker open until a template is chosen", async () => {
  const root = mount()
  await select("Scroll Fragment")
  const field = container.querySelector('[data-field-path="currency.items.Scroll Fragment.sound"]')!
  const type = field.querySelector<HTMLSelectElement>("select")!
  expect(type.value).toBe("none")
  type.value = "preset"
  type.dispatchEvent(new Event("change", { bubbles: true }))
  await nextTick()
  expect(type.value).toBe("preset")
  const templates = field.querySelectorAll<HTMLSelectElement>("select")[1]
  expect(templates.value).toBe("")
  await setCurrencyVisibility("Show") // An unrelated parent update must not close the editor.
  expect(type.value).toBe("preset")
  templates.value = "chromatic_orb"
  templates.dispatchEvent(new Event("change", { bubbles: true }))
  await nextTick()
  expect(getPath(root, "currency.items.Scroll Fragment.sound")).toEqual({ enabled: true, tts: "chromatic_orb" })
})

it("resyncs sound controls after external changes and restores the actual default on reset", async () => {
  const root = mount()
  await select("Chromatic Orb")
  const field = container.querySelector('[data-field-path="currency.items.Chromatic Orb.sound"]')!
  const type = field.querySelector<HTMLSelectElement>("select")!
  type.value = "tts"
  type.dispatchEvent(new Event("change", { bubbles: true }))
  await nextTick()
  expect(type.value).toBe("tts")
  const currency = root.currency as { items: Record<string, { sound: unknown }> }
  currency.items["Chromatic Orb"].sound = { enabled: true, soundId: 6 }
  await nextTick()
  expect(type.value).toBe("alert")
  expect(field.querySelectorAll<HTMLSelectElement>("select")[1].value).toBe("6")
  field.querySelector<HTMLButtonElement>('button[title="Use default"]')!.click()
  await nextTick()
  expect(type.value).toBe("preset")
  expect(field.querySelectorAll<HTMLSelectElement>("select")[1].value).toBe("chromatic_orb")
  expect(getPath(root, "currency.items.Chromatic Orb.sound")).toBeUndefined()
})

it("preserves the built-in TTS over tier sound and can replace it with an item-specific game sound", async () => {
  const root = mount({ currency: { tiers: { a: { sound: { tts: "Inherited TTS" } } } } })
  await select("Chaos Orb")
  const sound = container.querySelector<HTMLSelectElement>('[data-field-path="currency.items.Chaos Orb.sound"] select')!
  expect(sound.value).toBe("preset")
  sound.value = "alert"
  sound.dispatchEvent(new Event("change", { bubbles: true }))
  await nextTick()
  expect(sound.value).toBe("alert")
  expect(getPath(root, "currency.items.Chaos Orb.sound")).toEqual({ enabled: true, soundId: 1 })
})

it("groups league lists with every item selected by default, preserves an empty selection and resets it", async () => {
  container = document.createElement("div")
  document.body.append(container)
  const config = reactive({ buildProfile: {}, buildSpecificOptions: {} })
  app = createApp(ConfigForm, {
    modelValue: config,
    reference: { ...builtinReference, styles: {}, sounds: [] } as ReferenceData,
    filterName: "grouped-league",
  })
  app.mount(container)
  const league = container.querySelector('[data-editor-group="leagueMechanic"]')!
  expect(league.textContent).toContain("Covens of Umbra")
  expect(league.textContent).not.toContain("Chart room/reward")
  for (const group of ["viscera", "poppet"]) {
    expect(league.querySelector(`[data-field-path="leagueMechanic.rules.${group}.baseTypes"]`)).not.toBeNull()
  }
  expect(league.querySelector('[data-field-path^="leagueMechanic.rules.coralForest."]')).toBeNull()
  expect(config.buildSpecificOptions).toEqual({})
  const charts = league.querySelector('[data-field-path="leagueMechanic.rules.viscera.baseTypes"]')!
  const count = LEAGUE_RULES.find((rule) => rule.key === "viscera")!.baseTypes.length
  expect(charts.querySelectorAll('button[aria-pressed="true"]')).toHaveLength(count)
  expect(charts.querySelectorAll('button[aria-label^="Remove "]')).toHaveLength(0)
  const chart = charts.querySelector<HTMLButtonElement>('button[aria-label="Scorched Arm"]')!
  chart.click()
  await nextTick()
  expect(chart.getAttribute("aria-pressed")).toBe("false")
  expect(chart.textContent).toContain("Hidden")
  expect(charts.querySelectorAll("button[aria-pressed]")).toHaveLength(count)
  chart.click()
  await nextTick()
  expect(chart.getAttribute("aria-pressed")).toBe("true")
  expect(chart.textContent).not.toContain("Hidden")
  expect(getPath(config.buildSpecificOptions, "leagueMechanic.rules.viscera.baseTypes")).toBeUndefined()
  for (const toggle of charts.querySelectorAll<HTMLButtonElement>('button[aria-pressed="true"]')) {
    toggle.click()
    await nextTick()
  }
  expect(getPath(config.buildSpecificOptions, "leagueMechanic.rules.viscera.baseTypes")).toEqual([])
  expect(charts.querySelectorAll('button[aria-pressed="false"]')).toHaveLength(count)
  chart.click()
  await nextTick()
  expect(getPath(config.buildSpecificOptions, "leagueMechanic.rules.viscera.baseTypes")).toEqual(["Scorched Arm"])
  charts.querySelector<HTMLButtonElement>('button[title="Use default"]')!.click()
  await nextTick()
  expect(getPath(config.buildSpecificOptions, "leagueMechanic.rules.viscera.baseTypes")).toBeUndefined()
  expect(charts.querySelectorAll('button[aria-pressed="true"]')).toHaveLength(count)
  expect(charts.textContent).not.toContain("×")
  const currency = container.querySelector('[data-editor-group="currency"]')!
  expect(currency.querySelector('[data-field-path$=".style"]')).toBeNull()
  expect(currency.querySelector('[data-field-path="currency.gold.highStack.style"]')).toBeNull()
  expect(currency.querySelectorAll("summary")).toHaveLength(0)
  expect(league.querySelectorAll("summary")).toHaveLength(0)
  expect(league.querySelector('[data-field-path="leagueMechanic.style"]')).toBeNull()
  expect(league.querySelector('[data-field-path^="leagueMechanic.rules."][data-field-path$=".style"]')).toBeNull()
  expect(league.querySelector('[data-field-path^="leagueMechanic.rules."][data-field-path$=".sound"]')).toBeNull()
  expect(currency.textContent).not.toContain("Styles are managed in Global Settings")
  expect(charts.parentElement?.classList.contains("sm:col-span-2")).toBe(true)
  expect(league.querySelector('[data-field-path="leagueMechanic.rules.fated.visibility"]')).not.toBeNull()
})

it("starts Gold and scrolls collapsed without visibility controls", async () => {
  container = document.createElement("div")
  document.body.append(container)
  localStorage.setItem(
    "filtertool:collapsed:simple-currency",
    JSON.stringify({ "currency": false, "currency:currency.gold": true, "currency:currency.scrolls.wisdom": true }),
  )
  const config = reactive({ buildProfile: {}, buildSpecificOptions: {} })
  app = createApp(ConfigForm, {
    modelValue: config,
    reference: { ...builtinReference, styles: {}, sounds: [] } as ReferenceData,
    filterName: "simple-currency",
    defaults: { currency: { gold: { visibility: "Hide" } } },
  })
  app.mount(container)
  for (const [title, path] of [
    ["Gold", "currency.gold"],
    ["Scroll of Wisdom", "currency.scrolls.wisdom"],
    ["Portal Scroll", "currency.scrolls.portal"],
  ]) {
    expect(container.querySelector(`select[aria-label="${title} visibility"]`)).toBeNull()
    const header = container.querySelector<HTMLButtonElement>(`[data-editor-subsection="${path}"] button[aria-expanded]`)!
    expect(header.querySelector("span")?.textContent).toBe(title)
    expect(header.getAttribute("aria-expanded")).toBe("false")
    header.click()
    await nextTick()
  }
  for (const path of [
    "currency.gold.minStackSize",
    "currency.gold.stackSize",
    "currency.gold.highStackSize",
    "currency.scrolls.wisdom.soundCutoffLevel",
    "currency.scrolls.portal.soundCutoffLevel",
  ]) {
    const field = container.querySelector(`[data-field-path="${path}"]`)!
    expect(field).not.toBeNull()
    expect(field.closest('[style*="display: none"]')).toBeNull()
  }
  const currency = container.querySelector('[data-editor-group="currency"]')!
  for (const path of ["currency.scrolls.wisdom.soundCutoffLevel", "currency.scrolls.portal.soundCutoffLevel"]) {
    expect(container.querySelector(`[data-field-path="${path}"]`)?.parentElement?.classList.contains("sm:col-span-2")).toBe(true)
  }
  const minimum = container.querySelector('[data-field-path="currency.gold.minStackSize"]')!
  const stack = container.querySelector('[data-field-path="currency.gold.stackSize"]')!
  const highStack = container.querySelector('[data-field-path="currency.gold.highStackSize"]')!
  expect(minimum.parentElement?.parentElement).toBe(stack.parentElement?.parentElement)
  expect(stack.parentElement?.parentElement).toBe(highStack.parentElement?.parentElement)
  expect(minimum.parentElement?.classList.contains("sm:col-span-2")).toBe(false)
  expect(stack.parentElement?.classList.contains("sm:col-span-2")).toBe(false)
  expect(highStack.parentElement?.classList.contains("sm:col-span-2")).toBe(true)
  expect(currency.querySelectorAll("summary")).toHaveLength(0)
  expect(currency.querySelector('[data-field-path="currency.gold.sound"]')).toBeNull()
  expect(currency.querySelector('[data-field-path="currency.scrolls.wisdom.icon"]')).toBeNull()
})

it("distinguishes inherited icons and sounds from explicitly disabled callouts", async () => {
  container = document.createElement("div")
  document.body.append(container)
  const tier = BUILD_SPECIFIC_SCHEMA.find((group) => group.key === "currency")!.subsections![1]
  const field = tier.subsections!.find((section) => section.key.endsWith(".sound"))!.fields[0]
  let sound: unknown
  app = createApp(FieldInput, {
    field,
    "value": { enabled: true },
    "options": [],
    "onUpdate:value": (value: unknown) => {
      sound = value
    },
  })
  app.mount(container)
  const select = container.querySelector("select")!
  expect(select.selectedOptions[0].textContent).toContain("Existing per-item callout")
  select.value = "none"
  select.dispatchEvent(new Event("change", { bubbles: true }))
  await nextTick()
  expect(sound).toEqual({ enabled: false })
  app.unmount()
  const icon = tier.subsections!.find((section) => section.key.endsWith(".icon"))!.fields[0]
  let iconValue: unknown
  app = createApp(FieldInput, {
    "field": icon,
    "value": {},
    "options": [],
    "onUpdate:value": (value: unknown) => {
      iconValue = value
    },
  })
  app.mount(container)
  expect(container.textContent).toContain("Existing per-item icon")
  container.querySelector<HTMLButtonElement>('button[title="No Icon"]')!.click()
  await nextTick()
  expect(iconValue).toMatchObject({ enabled: false })
})
