// @vitest-environment jsdom
import { afterEach, expect, it } from "vitest"
import { createApp, nextTick, reactive, type App } from "vue"
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
  expect(fieldDefault(field("early.rules.armour2.size"), {})).toBe(40)
  expect(fieldDefault(field("early.rules.armour3.maxAreaLevel"), { early: { earlyMaxAreaLevel: 21 } })).toBe(21)
  expect(fieldDefault(field("links.rules.3generic.maxAreaLevel"), { links: { threeLinkMaxAreaLevel: 37 } })).toBe(37)
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
  expect([...root.querySelectorAll<HTMLInputElement>('input[type="color"]')].slice(0, 3).map((input) => input.value)).toEqual([
    "#c8c8c8",
    "#000000",
    "#000000",
  ])
  const preview = root.querySelector<HTMLElement>(".border[style]")!
  expect(preview.style.color).toBe("rgb(200, 200, 200)")
  expect(preview.style.backgroundColor).toBe("rgba(0, 0, 0, 0.941)")
})
