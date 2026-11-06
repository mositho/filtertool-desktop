// @vitest-environment jsdom
import { afterEach, expect, it } from "vitest"
import { createApp, h, nextTick, reactive, ref, type App } from "vue"
import ConfigForm from "../../frontend/src/components/ConfigForm.vue"
import StyleEditor from "../../frontend/src/components/StyleEditor.vue"
import { fieldDefault } from "../../frontend/src/fieldDefaults"
import {
  BUILD_SPECIFIC_SCHEMA,
  DEFAULTS_SCHEMA,
  filterEditorGroup,
  type FieldGroup,
  type FieldSubsection,
  type SchemaField,
} from "../../src/config/form-schema"
import reference from "../../internal/data/reference.json"
import type { ReferenceData } from "../../frontend/src/api"
import { editColor } from "./colorFields"
import { DEFAULT_STYLE_SETTINGS } from "../../src/style-defaults"
import { styleSwatch } from "../../frontend/src/namedStyles"

let app: App
let root: HTMLDivElement
afterEach(() => {
  app?.unmount()
  root?.remove()
  localStorage.clear()
})
const refData = { ...reference, styles: {}, sounds: [], styleNames: [] } as unknown as ReferenceData
function mount(component: typeof ConfigForm | typeof StyleEditor, props: Record<string, unknown>) {
  root = document.createElement("div")
  document.body.append(root)
  app = createApp(component as typeof ConfigForm, props)
  app.mount(root)
}
function fieldsOf(groups: FieldGroup[]): SchemaField[] {
  const nested = (section: FieldSubsection): SchemaField[] => [...section.fields, ...(section.subsections ?? []).flatMap(nested)]
  return groups.flatMap((group) => [...group.fields, ...(group.subsections ?? []).flatMap(nested)])
}
const fields = fieldsOf(BUILD_SPECIFIC_SCHEMA)
const field = (path: string) => fields.find((entry) => entry.path === path)!

it("keeps all built-in rule tuning global, with only paired maximum and size fields", () => {
  const global = fieldsOf(DEFAULTS_SCHEMA).filter((field) => /^(early|links|normalItems|magicItems|rareItems)\.rules\./.test(field.path))
  expect(global).toHaveLength(58)
  expect(global.every((field) => field.control === "number" && /\.(maxAreaLevel|size)$/.test(field.path))).toBe(true)
  const local = fieldsOf(BUILD_SPECIFIC_SCHEMA.map(filterEditorGroup))
  expect(local.some((field) => /^(early|links|normalItems|magicItems|rareItems)\.rules\./.test(field.path))).toBe(false)
})

it("removes Advanced Rules from filters while retaining ordinary cutoffs and custom highlights", () => {
  mount(ConfigForm, { modelValue: { buildProfile: {}, buildSpecificOptions: {} }, filterName: "test", reference: refData })
  expect(root.textContent).not.toContain("Advanced Rules")
  for (const path of [
    "early.earlyMaxAreaLevel",
    "links.threeLinkMaxAreaLevel",
    "normalItems.maxAreaLevel",
    "links.genericThreeLinkMaxAreaLevel",
  ]) {
    expect(root.querySelector(`[data-field-path="${path}"]`)).not.toBeNull()
  }
  expect(root.textContent).toContain("Highlights")
  // League selections are not built-in rule customization.
  expect(
    [...root.querySelectorAll('[data-field-path*=".rules."]')].every((entry) =>
      entry.getAttribute("data-field-path")?.startsWith("leagueMechanic."),
    ),
  ).toBe(true)
})

it("inherits global rule size from the shared preset and global rule maxima from the regular cutoffs", () => {
  const size = field("links.rules.3generic.size")
  expect(fieldDefault(size, {}, {}, { threeLink: { size: 28 } })).toBe(28)
  expect(fieldDefault(size, {}, { links: { rules: { "3generic": { size: 31 } } } }, { threeLink: { size: 28 } })).toBe(31)
  const socketSize = field("early.rules.armour2.size")
  expect(fieldDefault(socketSize, {})).toBe(socketSize.defaultValue)
  expect(fieldDefault(field("early.rules.armour3.maxAreaLevel"), { early: { earlyMaxAreaLevel: 21 } })).toBe(21)
  expect(fieldDefault(field("links.rules.3generic.maxAreaLevel"), { links: { threeLinkMaxAreaLevel: 37 } })).toBe(37)
  for (const key of ["1x1", "1x3", "2x2", "2x4"]) {
    expect(fieldDefault(field(`magicItems.rules.${key}.maxAreaLevel`), { magicItems: { maxAreaLevel: 31 } })).toBe(31)
  }
})

it("separates size falloff minima from visibility maxima and exposes one magic cutoff", () => {
  const equipment = BUILD_SPECIFIC_SCHEMA.find((group) => group.key === "equipment")!
  const maximum = equipment.subsections!.find((section) => section.title === "Max Area Level")!
  const minimum = equipment.subsections!.find((section) => section.title === "Min Area Level")!
  expect(maximum.fields.every((entry) => !entry.path.endsWith("footprintMinAreaLevel"))).toBe(true)
  expect(maximum.fields.filter((entry) => entry.path.startsWith("magicItems.")).map((entry) => entry.path)).toEqual([
    "magicItems.maxAreaLevel",
  ])
  expect(minimum.fields.map((entry) => entry.label)).toEqual(["Magic Items · Size Falloff", "Rare Items · Size Falloff"])
  expect(minimum.fields.every((entry) => entry.tooltip === "From this area level, size depends on the inventory space of the item.")).toBe(
    true,
  )
  const early = BUILD_SPECIFIC_SCHEMA.find((group) => group.key === "early")!
  expect(early.subsections![0].title).toBe("Max Area Level")
  expect(field("early.earlyMaxAreaLevel").label).toBe("Generic")
  expect(field("early.shieldMaxAreaLevel").label).toBe("Shields")
})

it("keeps linked early cutoffs dynamic and restores inherited filter values", async () => {
  const modelValue = reactive({
    buildProfile: {},
    buildSpecificOptions: { early: { earlyMaxAreaLevel: 21, threeSocketMaxAreaLevel: 16 as number | undefined } },
  })
  mount(ConfigForm, { modelValue, filterName: "test", reference: refData })
  const row = root.querySelector('[data-field-path="early.threeSocketMaxAreaLevel"]')!
  expect(row.querySelector("input")?.value).toBe("16")
  row.querySelector<HTMLButtonElement>('button[title="Use default"]')!.click()
  await nextTick()
  expect(modelValue.buildSpecificOptions.early.threeSocketMaxAreaLevel).toBeUndefined()
  expect(row.querySelector("input")?.placeholder).toBe("21")
  modelValue.buildSpecificOptions.early.earlyMaxAreaLevel = 25
  await nextTick()
  expect(row.querySelector("input")?.placeholder).toBe("25")
})

it("exposes independent footprint minima and restores the linked rare minimum", async () => {
  expect(fieldDefault(field("magicItems.footprintMinAreaLevel"), {})).toBe(0)
  expect(fieldDefault(field("rareItems.footprintMinAreaLevel"), { early: { earlyMaxAreaLevel: 21 } })).toBe(21)
  const modelValue = reactive({
    buildProfile: {},
    buildSpecificOptions: {
      early: { earlyMaxAreaLevel: 21 },
      rareItems: { footprintMinAreaLevel: 8 as number | undefined },
    },
  })
  mount(ConfigForm, { modelValue, filterName: "footprint-minimum", reference: refData })
  expect(root.querySelector('[data-field-path="magicItems.footprintMinAreaLevel"]')).not.toBeNull()
  const row = root.querySelector('[data-field-path="rareItems.footprintMinAreaLevel"]')!
  expect(row.querySelector("input")?.value).toBe("8")
  row.querySelector<HTMLButtonElement>('button[title="Use default"]')!.click()
  await nextTick()
  expect(modelValue.buildSpecificOptions.rareItems.footprintMinAreaLevel).toBeUndefined()
  expect(row.querySelector("input")?.placeholder).toBe("21")
  modelValue.buildSpecificOptions.early.earlyMaxAreaLevel = 25
  await nextTick()
  expect(row.querySelector("input")?.placeholder).toBe("25")
})

it("labels an inherited uncapped rare-item limit as using default and restores it after resetting an override", async () => {
  const modelValue = reactive({ buildProfile: {}, buildSpecificOptions: {} as Record<string, unknown> })
  mount(ConfigForm, { modelValue, filterName: "rare-default", reference: refData, defaults: { rareItems: {} } })
  const row = root.querySelector('[data-field-path="rareItems.maxAreaLevel"]')!
  const input = row.querySelector<HTMLInputElement>("input")!
  expect(input.placeholder).toBe("uncapped")
  expect(row.textContent).toContain("using default")
  expect(modelValue.buildSpecificOptions).toEqual({})
  input.value = "45"
  input.dispatchEvent(new Event("input", { bubbles: true }))
  await nextTick()
  expect(row.textContent).not.toContain("using default")
  row.querySelector<HTMLButtonElement>('[title="Use default"]')!.click()
  await nextTick()
  expect(input.value).toBe("")
  expect(input.placeholder).toBe("uncapped")
  expect(row.textContent).toContain("using default")
})

it("recognizes a saved default shield progression and clears redundant overrides when reselected", async () => {
  const modelValue = reactive({ buildProfile: { shieldProgression: "early" as string | undefined }, buildSpecificOptions: {} })
  const defaults = reactive({ shieldProgression: "early" })
  mount(ConfigForm, { modelValue, filterName: "shield-default", reference: refData, defaults })
  const row = root.querySelector('[data-field-path="shieldProgression"]')!
  const select = row.querySelector<HTMLSelectElement>("select")!
  expect(select.value).toBe("early")
  expect(row.textContent).toContain("using default")
  defaults.shieldProgression = "full"
  await nextTick()
  expect(row.textContent).not.toContain("using default")
  defaults.shieldProgression = "early"
  select.value = "full"
  select.dispatchEvent(new Event("change", { bubbles: true }))
  await nextTick()
  expect(row.textContent).not.toContain("using default")
  select.value = "early"
  select.dispatchEvent(new Event("change", { bubbles: true }))
  await nextTick()
  expect(modelValue.buildProfile.shieldProgression).toBeUndefined()
  expect(row.textContent).toContain("using default")
})

it("places the early generic-link cutoff before toggles on its own row", () => {
  mount(ConfigForm, { modelValue: { buildProfile: {}, buildSpecificOptions: {} }, filterName: "test", reference: refData })
  const wrapper = root.querySelector('[data-field-path="links.genericThreeLinkMaxAreaLevel"]')!.parentElement!
  expect(wrapper.classList.contains("sm:col-span-2")).toBe(true)
  expect(wrapper.parentElement!.firstElementChild).toBe(wrapper)
})

it("keeps starter supports out of style-only choices but allows spoken callouts", async () => {
  const modelValue = {
    buildProfile: {},
    buildSpecificOptions: { gemCallouts: { styleOnly: ["Momentum Support", "Added Cold Damage Support"], gems: ["Momentum Support"] } },
  }
  mount(ConfigForm, {
    modelValue,
    filterName: "test",
    reference: { ...refData, supportGems: ["Momentum Support", "Added Cold Damage Support"] },
  })
  const choices = root.querySelector('[data-field-path="gemCallouts.styleOnly"]')!
  expect(choices.textContent).not.toContain("Momentum Support")
  expect(choices.textContent).toContain("Added Cold Damage Support")
  expect(root.querySelector('[data-field-path="gemCallouts.gems"]')!.textContent).toContain("Momentum Support")
  choices.querySelector<HTMLElement>('[role="group"]')!.click()
  await nextTick()
  expect(choices.querySelector("ul")!.textContent).not.toContain("Momentum Support")
  expect(modelValue.buildSpecificOptions.gemCallouts.styleOnly).toEqual(["Momentum Support", "Added Cold Damage Support"])
})

it("previews unspecified style colors using compiler defaults", () => {
  mount(StyleEditor, { modelValue: {}, styleNames: [], styles: {} })
  expect(
    [...root.querySelectorAll<HTMLButtonElement>('button[aria-haspopup="dialog"][value]')].slice(0, 3).map((input) => input.value),
  ).toEqual(
    [DEFAULT_STYLE_SETTINGS.text, DEFAULT_STYLE_SETTINGS.background, DEFAULT_STYLE_SETTINGS.border].map((color) => color.toLowerCase()),
  )
  const preview = root.querySelector<HTMLElement>(".border[style]")!
  const expected = document.createElement("div")
  Object.assign(expected.style, styleSwatch(DEFAULT_STYLE_SETTINGS))
  expect(preview.style.color).toBe(expected.style.color)
  expect(preview.style.backgroundColor).toBe(expected.style.backgroundColor)
})

it("edits inline colors with the shared picker and detaches from the preset without losing its other properties", async () => {
  const value = ref<Record<string, unknown>>({ preset: "example" })
  root = document.createElement("div")
  document.body.append(root)
  const style = { text: "#abcdef", background: "#102030", border: "#405060", size: 35, backgroundOpacity: 0.5 }
  app = createApp({
    setup: () => () =>
      h(StyleEditor, {
        "modelValue": value.value,
        "styles": { example: style },
        "styleNames": ["example"],
        "onUpdate:modelValue": (next) => (value.value = next ?? {}),
      }),
  })
  app.mount(root)
  await editColor(root.querySelector<HTMLButtonElement>('[aria-label="Style text"]')!, "#123456")
  expect(value.value).toEqual({ ...style, text: "#123456" })
  expect(root.querySelector<HTMLSelectElement>("select")!.value).toBe("")
})

it("labels currency presets with the editor tiers without changing their saved values", () => {
  let saved: Record<string, unknown> | undefined
  mount(StyleEditor, {
    "modelValue": { preset: "currencyB" },
    "styleNames": ["currencyA", "currencyB", "currencyC"],
    "styles": {},
    "onUpdate:modelValue": (value: Record<string, unknown> | undefined) => (saved = value),
  })
  const select = root.querySelector<HTMLSelectElement>("select")!
  expect([...select.options].slice(1).map((option) => [option.value, option.textContent])).toEqual([
    ["currencyA", "Currency · S Tier"],
    ["currencyB", "Currency · A Tier"],
    ["currencyC", "Currency · B Tier"],
  ])
  expect(select.value).toBe("currencyB")
  expect(root.textContent).toContain("Currency · A Tier")
  select.value = "currencyC"
  select.dispatchEvent(new Event("change", { bubbles: true }))
  expect(saved).toEqual({ preset: "currencyC" })
})
